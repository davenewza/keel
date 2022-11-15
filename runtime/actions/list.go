package actions

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/samber/lo"
	"github.com/teamkeel/keel/proto"
)

type ListResult struct {
	Results     []map[string]any `json:"results"`
	HasNextPage bool             `json:"hasNextPage"`
}

func applyImplicitFiltersForList(scope *Scope, args WhereArgs) error {
	allJoins := []string{}

inputs:
	for _, input := range scope.operation.Inputs {
		if input.Behaviour != proto.InputBehaviour_INPUT_BEHAVIOUR_IMPLICIT {
			continue
		}

		fieldName := input.Name
		value, ok := args[fieldName]

		// not found
		if !ok {
			if input.Optional {
				continue inputs
			}

			return fmt.Errorf("did not find required '%s' input in where clause", fieldName)
		}

		valueMap, ok := value.(map[string]any)

		if !ok {
			if input.Optional {
				// do not do any further processing if the input is not a map
				// as it is likely nil
				continue inputs
			}

			return fmt.Errorf("'%s' input value %v to not in correct format", fieldName, value)
		}

		for operatorStr, operand := range valueMap {
			operator, err := graphQlOperatorToActionOperator(operatorStr)
			if err != nil {
				return err
			}

			// New filter resolver to generate a database query statement
			resolver := NewImplicitFilterResolverResolver(scope)

			// Resolve the database statement for this expression
			statement, joins, err := resolver.ResolveQueryStatement(input, fieldName, operand, operator)
			if err != nil {
				return err
			}

			allJoins = append(allJoins, joins...)

			scope.query = scope.query.
				WithContext(scope.context).
				Where(statement)
		}
	}

	allJoins = lo.Uniq(allJoins)
	scope.query = scope.query.Joins(strings.Join(allJoins, " "))

	return nil
}

func List(scope *Scope, input map[string]any) (*ListResult, error) {
	where, ok := input["where"].(map[string]any)
	if !ok {
		where = map[string]any{}
	}

	err := applyImplicitFiltersForList(scope, where)
	if err != nil {
		return nil, err
	}

	err = DefaultApplyExplicitFilters(scope, where)
	if err != nil {
		return nil, err
	}

	isAuthorised, err := DefaultIsAuthorised(scope, where)
	if err != nil {
		return nil, err
	}

	if !isAuthorised {
		return nil, errors.New("not authorized to access this operation")
	}

	op := scope.operation

	if op.Implementation == proto.OperationImplementation_OPERATION_IMPLEMENTATION_CUSTOM {
		// TODO: the custom function should receive the whole input, not just the
		// where's
		return ParseListResponse(scope.context, op, where)
	}

	page, err := parsePage(input)
	if err != nil {
		return nil, err
	}

	// Specify the ORDER BY - but also a "LEAD" extra column to harvest extra data
	// that helps to determine "hasNextPage".
	by := fmt.Sprintf("%s.id", strcase.ToSnake(scope.model.Name))

	selectArgs := `DISTINCT ON (%[1]s.id) 
		%[1]s.*,
		CASE WHEN lead(%[1]s.id) OVER ( order by %[1]s.id ) is not null THEN true ELSE false END as hasNext
		`
	selectArgs = fmt.Sprintf(selectArgs, strcase.ToSnake(scope.model.Name))

	scope.query = scope.query.WithContext(scope.context).Select(selectArgs, by)
	scope.query = scope.query.WithContext(scope.context).Order(by)

	// A Where clause to implement the after/before paging request
	switch {
	case page.After != "":
		scope.query = scope.query.WithContext(scope.context).Where("ID > ?", page.After)
	case page.Before != "":
		scope.query = scope.query.WithContext(scope.context).Where("ID < ?", page.Before)
	}

	switch {
	case page.First != 0:
		scope.query = scope.query.WithContext(scope.context).Limit(page.First)
	case page.Last != 0:
		scope.query = scope.query.WithContext(scope.context).Limit(page.Last)
	}

	// Execute the query
	result := []map[string]any{}
	err = scope.query.WithContext(scope.context).Find(&result).Error
	if err != nil {
		return nil, err
	}

	// Sort out the hasNextPage value, and clean up the response.
	hasNextPage := false
	if len(result) > 0 {
		last := result[len(result)-1]
		hasNextPage = last["hasnext"].(bool)
	}

	for _, row := range result {
		delete(row, "has_next")
	}

	return &ListResult{
		Results:     toLowerCamelMaps(result),
		HasNextPage: hasNextPage,
	}, nil
}

// parsePage extracts page mandate information from the given map and uses it to
// compose a Page.
func parsePage(args map[string]any) (Page, error) {
	page := Page{}

	if first, ok := args["first"]; ok {
		asInt, ok := first.(int)
		if !ok {
			var err error
			asInt, err = strconv.Atoi(first.(string))
			if err != nil {
				return page, fmt.Errorf("cannot cast this: %v to an int", first)
			}
		}
		page.First = asInt
	}

	if last, ok := args["last"]; ok {
		asInt, ok := last.(int)
		if !ok {
			var err error
			asInt, err = strconv.Atoi(last.(string))
			if err != nil {
				return page, fmt.Errorf("cannot cast this: %v to an int", last)
			}
		}
		page.Last = asInt
	}

	if after, ok := args["after"]; ok {
		asString, ok := after.(string)
		if !ok {
			return page, fmt.Errorf("cannot cast this: %v to a string", after)
		}
		page.After = asString
	}

	if before, ok := args["before"]; ok {
		asString, ok := before.(string)
		if !ok {
			return page, fmt.Errorf("cannot cast this: %v to a string", before)
		}
		page.Before = asString
	}

	// If none specified - use a sensible default
	if page.First == 0 && page.Last == 0 {
		page = Page{First: 50}
	}

	return page, nil
}

// A Page describes which page you want from a list of records,
// in the style of this "Connection" pattern:
// https://relay.dev/graphql/connections.htm
//
// Consider for example, that you previously fetched a page of 10 records
// and from that previous response you also knew that the last of those 10 records
// could be referred to with the opaque cursor "abc123". Armed with that information you can
// ask for the next page of 10 records by setting First to 10, and After to "abc123".
//
// To move backwards, you'd set the Last and Before fields instead.
//
// When you have no prior positional context you should specify First but leave Before and After to
// the empty string. This gives you the first N records.
type Page struct {
	First  int
	Last   int
	After  string
	Before string
}
