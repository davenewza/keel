declare module "@teamkeel/sdk/constraints" {
  export type StringConstraint =
    | string
    | {
        startsWith?: string;
        endsWith?: string;
        oneOf?: string[];
        contains?: string;
        notEquals?: string;
        equals?: string;
      };
  export type NumberConstraint =
    | number
    | {
        greaterThan?: number;
        greaterThanOrEquals?: number;
        lessThan?: number;
        lessThanOrEquals?: number;
        equals?: number;
        notEquals?: number;
      };
  export type DateConstraint =
    | Date
    | {
        equal?: Date;
        before?: Date;
        onOrBefore?: Date;
        after?: Date;
        onOrAfter?: Date;
      };
  export type BooleanConstraint = boolean | {
    notEquals?: boolean;
    equals?: boolean;
  };
  export type EnumConstraint = string | {
    notEquals?: string;
    equals?: string;
  };
}
declare module "@teamkeel/sdk/index" {
  import Query, { ChainableQuery } from "@teamkeel/sdk/query";
  import * as QueryConstraints from "@teamkeel/sdk/constraints";
  import Logger, {
    ConsoleTransport,
    Level as LogLevel,
  } from "@teamkeel/sdk/logger";
  import { Identity } from "@teamkeel/sdk/types";
  import { queryResolverFromEnv } from "@teamkeel/sdk/db/resolver";

  //@ts-ignore
  export * from "@teamkeel/client";

  export * from "@teamkeel/sdk/returnTypes";

  export {
    Query,
    ChainableQuery,
    QueryConstraints,
    Logger,
    ConsoleTransport,
    LogLevel,
    Identity,
    queryResolverFromEnv,
  };
}

declare module "@teamkeel/sdk/logger" {
  export enum Level {
    Info = "info",
    Error = "error",
    Debug = "debug",
    Warn = "warn",
    Success = "success"
  }
  type Msg = any;

  export interface Transport {
    log: (msg: Msg, level: Level, options: LoggerOptions) => void;
  }
  export interface LoggerOptions {
    transport?: Transport;
    colorize?: boolean;
    timestamps?: boolean;
  }

  export class ConsoleTransport implements Transport {
    log: (msg: Msg, level: Level, options: LoggerOptions) => void;
  }

  export default class Logger {
    private readonly options: LoggerOptions;

    constructor(opts?: LoggerOptions);

    log: (msg: Msg, level: Level) => void;
  }
}

declare module "@teamkeel/sdk/query" {
  import {
    Conditions,
    ChainedQueryOpts,
    QueryOpts,
    Input,
    OrderClauses,
  } from "@teamkeel/sdk/types";
  import * as ReturnTypes from "@teamkeel/sdk/returnTypes";
  import { QueryResolver } from "@teamkeel/sdk/db/resolver";
  export class ChainableQuery<T> {
    private readonly tableName;
    private readonly conditions;
    private readonly queryResolver: QueryResolver;
    constructor({
      tableName,
      queryResolver,
      conditions,
    }: ChainedQueryOpts<T>);
    orWhere: (conditions: Conditions<T>) => ChainableQuery<T>;
    all: () => Promise<ReturnTypes.FunctionListResponse<T>>;
    order: (clauses: OrderClauses<T>) => ChainableQuery<T>;
    findOne: () => Promise<ReturnTypes.FunctionGetResponse<T>>;
    private appendConditions;
  }
  export default class Query<T> {
    private readonly tableName;
    private readonly conditions;
    private orderClauses;
    private readonly queryResolver: QueryResolver;

    constructor({ tableName, queryResolver, logger }: QueryOpts);
    create: (
      inputs: Partial<T>
    ) => Promise<ReturnTypes.FunctionCreateResponse<T>>;
    where: (conditions: Conditions<T>) => ChainableQuery<T>;
    delete: (id: string) => Promise<ReturnTypes.FunctionDeleteResponse<T>>;
    findOne: (
      conditions: Conditions<T>
    ) => Promise<ReturnTypes.FunctionGetResponse<T>>;
    update: (
      id: string,
      inputs: Input<T>
    ) => Promise<ReturnTypes.FunctionUpdateResponse<T>>;
    all: () => Promise<ReturnTypes.FunctionListResponse<T>>;
  }
  
  export {};
}

declare module "@teamkeel/sdk/db/query" {
  export type SqlQueryParts = SqlQueryPart[];

  export type SqlQueryPart = RawSql | SqlInput;

  export type RawSql = {
    type: "sql";
    value: string;
  };

  export type SqlInput = {
    type: "input";
    value: any;
  };
}

declare module "@teamkeel/sdk/db/resolver" {
  import pg from "pg";
  import { SqlQueryParts } from "@teamkeel/sdk/db/query";
  export interface QueryResolver {
    runQuery(query: SqlQueryParts): Promise<QueryResult>;
  }

  export interface QueryResult {
    rows: QueryResultRow[];
  }

  export interface QueryResultRow {
    [column: string]: any;
  }
  export function queryResolverFromEnv(env: Record<string, string | undefined>): QueryResolver
  export class PgQueryResolver implements QueryResolver {
    private readonly pool: pg.Pool;
    constructor(config: { connectionString: string })

    runQuery(query: SqlQueryParts): Promise<QueryResult>
    private toQuery(query: SqlQueryParts): { text: string; values: any[] }
  }
}

declare module "@teamkeel/sdk/queryBuilders/index" {
  import { Conditions } from "@teamkeel/sdk/types";
  import { SqlQueryParts } from "@teamkeel/sdk/db/query";
  export const buildSelectStatement: <T>(
    tableName: string,
    conditions: Conditions<T>[]
  ) => SqlQueryParts;
  export const buildCreateStatement: <T>(
    tableName: string,
    inputs: Partial<T>
  ) => SqlQueryParts;
  export const buildUpdateStatement: <T>(
    tableName: string,
    id: string,
    inputs: Partial<T>
  ) => SqlQueryParts;
  export const buildDeleteStatement: <T>(
    tableName: string,
    id: string
  ) => SqlQueryParts;
}
declare module "@teamkeel/sdk/types" {
  import {
    StringConstraint,
    BooleanConstraint,
    NumberConstraint,
    DateConstraint,
    EnumConstraint,
  } from "@teamkeel/sdk/constraints";
  import { Logger } from "@teamkeel/sdk";
  import { QueryResolver } from "@teamkeel/sdk/db/resolver";
  export interface QueryOpts {
    tableName: string;
    queryResolver: QueryResolver;
    logger: Logger;
  }
  export interface ChainedQueryOpts<T> extends QueryOpts {
    conditions: Conditions<T>[];
  }
  export type Constraints<T> = T extends String
    ? StringConstraint
    : T extends Boolean
    ? BooleanConstraint
    : T extends Number
    ? NumberConstraint
    : T extends Date
    ? DateConstraint
    : EnumConstraint;
  export type Input<T> = Record<keyof T, unknown>;
  export type Conditions<T> = Partial<{ [K in keyof T]: Constraints<T[K]> }>;
  export type OrderDirection = "ASC" | "DESC";
  export type OrderClauses<T> = Partial<Record<keyof T, OrderDirection>>;

  // A generic Identity interface for usage in other npm packages
  // without codegenerating the whole Identity interface
  // We know that Identity will implement these fields
  export interface Identity {
    id: string;
    email: string;
  }
}

declare module "@teamkeel/sdk/returnTypes" {
  // ValidationErrors will be returned when interacting with
  // the Query API (creating, updating entities)
  export interface ValidationError {
    field: string;
    message: string;
    code: string;
  }

  // ExecutionError represents other misc errors
  // that can occur during the execution of a custom function
  export interface ExecutionError {
    message: string;

    // todo: implement stacks
    stack: string;
  }

  export type FunctionError = ValidationError | ExecutionError;

  export interface FunctionCreateResponse<T> {
    object?: T;
    errors?: FunctionError[];
  }

  export interface FunctionGetResponse<T> {
    object?: T;
    errors?: FunctionError[];
  }

  export interface FunctionDeleteResponse<T> {
    success: boolean;
    errors?: FunctionError[];
  }

  export interface FunctionListResponse<T> {
    collection: T[];
    errors?: FunctionError[];
    // todo: add type for pagination
  }

  export interface FunctionUpdateResponse<T> {
    object?: T;
    errors?: FunctionError[];
  }

  export interface FunctionAuthenticateResponse {
    identityId?: string;
    identityCreated: boolean;
    errors?: FunctionError[];
  }
}

declare module "@teamkeel/sdk" {
  import main = require("@teamkeel/sdk/index");
  export = main;
}