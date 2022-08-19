import { RunnerOpts, Test, TestFunc, TestName } from './types'
import { AssertionFailure } from './errors'
import { TestCaseResult, TestResult } from './output'
import { expect } from './expect'
import Reporter from './reporter'

// generated.ts doesnt exist at this point, but IT WILL 😈
// eslint-disable-next-line @typescript-eslint/ban-ts-comment
//@ts-ignore
export * from './generated'

const tests : Test[] = []

function test(testName: TestName, fn: TestFunc) {
  tests.push({
    testName,
    fn,
  })
}

// global - reset with every instantiation of module.
let results: TestCaseResult[] = []

async function runAllTests({ parentPort }: RunnerOpts) {
  const reporter = new Reporter({
    host: 'localhost',
    port: parentPort
  })
  results = []

  if (!tests.length) {
    return
  } 

  for (const { testName, fn } of tests) {
    let result : TestResult | undefined = undefined

    try {
      const t = fn()

      // support both async and non async invocations
      const isPromisified = t instanceof Promise

      // if we do not await the result of the func,
      // then the catch block will not catch the error
      if (isPromisified) {
        await t
      }

      result = TestResult.pass(testName)
    } catch (err) {      
      const isAssertionFailure = err instanceof AssertionFailure

      if (isAssertionFailure) {
        const { actual, expected } = err as AssertionFailure

        result = TestResult.fail(
          testName,
          actual,
          expected,
        )
      } else {
        result = TestResult.exception(testName, err as Error)
      }
    } finally {
      if (result) {
        results.push(result.asObject())
      }
    }
  }

  // report back to parent process with all
  // results for tests in the current file.
  reporter.report(results)
}

export {
  test,
  expect,
  runAllTests
}