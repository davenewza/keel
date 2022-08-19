// All of the different constraint types are unions of the underlying type
// or an object type which you can use to query by a set of permitted operators
// based on the type. e.g if you are querying a number field, then you can also perform number
// related operations on that field such as gte / lte etc
// Where the union resolves to the actual type such as string or number, this is equivalent
// to an equality check.

// sample query object:
// {
//   myStringField: "this is a string", // <== shorthand means "equal"
//   myNumberField: {
//     greaterThan: 10
//   }
//   myOtherNumberField: 10 // <== equality check
// }

export type EqualityConstraint = {
  notEqual?: string
  equal?: string
}

export type StringConstraint = string | {
  startsWith?: string
  endsWith?: string
  oneOf?: string[]
  contains?: string
  // todo: need to enforce usage of only one of 
  // equal or not equal - it is not a simple union of the two
} | EqualityConstraint

export type NumberConstraint = number | {
  greaterThan?: number
  greaterThanOrEqualTo?: number
  lessThan?: number
  lessThanOrEqualTo?: number
  // todo: need to enforce usage of only one of 
  // equal or not equal - it is not a simple union of the two
  equal?: number
  notEqual?: number
} | EqualityConstraint

export type BooleanConstraint =  EqualityConstraint

export type DateConstraint = Date
// todo:
// - Date
// - Enum
// - etc