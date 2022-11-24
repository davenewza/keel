import {
  StringConstraint,
  BooleanConstraint,
  NumberConstraint,
  DateConstraint,
  EnumConstraint,
} from "./constraints";
import { Logger } from "./";
import { QueryResolver } from "./db/resolver";

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

export interface BuiltInFields {
  id: string;
  createdAt: Date;
  updatedAt: Date;
}

export type OrderDirection = "ASC" | "DESC";
export type OrderClauses<T> = Partial<Record<keyof T, OrderDirection>>;

// A generic Identity interface for usage in other npm packages
// without codegenerating the whole Identity interface
// We know that Identity will implement these fields
// TODO: remove once we're codegenerating this from the schema.
export interface Identity {
  id: string;
  email: string;
}

export interface CustomFunction {
  call: any;
}

// Config represents the configuration values
// to be passed to the Custom Code runtime server
export interface Config {
  functions: Record<string, CustomFunction>;
  api: unknown;
}
