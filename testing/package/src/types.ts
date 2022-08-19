export type TestFunc = () => void;
export type TestName = string;

export interface Test {
  testName: TestName;
  fn: TestFunc;
}

export interface RunnerOpts {
  parentPort: number;
}