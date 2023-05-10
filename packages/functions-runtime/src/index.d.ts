export type IDWhereCondition = {
  equals?: string | null;
  oneOf?: string[] | null;
};

export type StringWhereCondition = {
  startsWith?: string | null;
  endsWith?: string | null;
  oneOf?: string[] | null;
  contains?: string | null;
  notEquals?: string | null;
  equals?: string | null;
};

export type BooleanWhereCondition = {
  equals?: boolean | null;
  notEquals?: boolean | null;
};

export type NumberWhereCondition = {
  greaterThan?: number | null;
  greaterThanOrEquals?: number | null;
  lessThan?: number | null;
  lessThanOrEquals?: number | null;
  equals?: number | null;
  notEquals?: number | null;
};

// Date database API
export type DateWhereCondition = {
  equals?: Date | string | null;
  before?: Date | string | null;
  onOrBefore?: Date | string | null;
  after?: Date | string | null;
  onOrAfter?: Date | string | null;
};

// Date query input
export type DateQueryInput = {
  equals?: string | null;
  before?: string | null;
  onOrBefore?: string | null;
  after?: string | null;
  onOrAfter?: string | null;
};

// Timestamp query input
export type TimestampQueryInput = {
  before: string | null;
  after: string | null;
};

export type ContextAPI = {
  headers: RequestHeaders;
  response: Response;
  isAuthenticated: boolean;
  now(): Date;
};

export type Response = {
  headers: Headers;
};

export type PageInfo = {
  startCursor: string;
  endCursor: string;
  totalCount: number;
  hasNextPage: boolean;
  count: number;
};

// Request headers query API
export type RequestHeaders = {
  get(name: string): string;
  has(name: string): boolean;
};

export declare class Permissions {
  constructor();

  // allow() can be used to explicitly permit access to an action
  allow(): void;

  // deny() can be used to explicitly deny access to an action
  deny(): never;
}
