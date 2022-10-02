import { test, expect, actions, Post } from '@teamkeel/testing'

test('text permission successful', async () => {
  const { errors } = await actions
    .createPostTextPermission({ title: "hello" })

  var authorizationFailed = hasAuthorizationError(errors)
  expect.equal(authorizationFailed, false)
})

test('text permission failed', async () => {
  const { errors } = await actions
    .createPostTextPermission({ title: "not hello" })

  var authorizationFailed = hasAuthorizationError(errors)
  expect.equal(authorizationFailed, true)
})

test('number permission successful', async () => {
  const { errors } = await actions
    .createPostNumberPermission({ views: 5 })

  var authorizationFailed = hasAuthorizationError(errors)
  expect.equal(authorizationFailed, false)
})

test('number permission failed', async () => {
  const { errors } = await actions
    .createPostNumberPermission({ views: 500 })

  var authorizationFailed = hasAuthorizationError(errors)
  expect.equal(authorizationFailed, true)
})

test('boolean permission successful', async () => {
  const { errors } = await actions
    .createPostBooleanPermission({ active: true })

  var authorizationFailed = hasAuthorizationError(errors)
  expect.equal(authorizationFailed, false)
})

test('boolean permission failed', async () => {
  const { errors } = await actions
    .createPostBooleanPermission({ active: false })

  var authorizationFailed = hasAuthorizationError(errors)
  expect.equal(authorizationFailed, true)
})

test('text not null permission successful', async () => {
  const { errors } = await actions
    .createPostTextNotNullPermission({ title: "hello" })

  var authorizationFailed = hasAuthorizationError(errors)
  expect.equal(authorizationFailed, false)
})

test('text not null permission failed', async () => {
  const { errors } = await actions
    .createPostTextNotNullPermission({ title: null })

  var authorizationFailed = hasAuthorizationError(errors)
  expect.equal(authorizationFailed, true)
})

test('text null permission failed', async () => {
  const { errors } = await actions
    .createPostTextNullPermission({})

  var authorizationFailed = hasAuthorizationError(errors)
  expect.equal(authorizationFailed, true)
})

test('number not null permission successful', async () => {
  const { errors } = await actions
    .createPostNumberNotNullPermission({ views: 5 })

  var authorizationFailed = hasAuthorizationError(errors)
  expect.equal(authorizationFailed, false)
})

test('number not null permission failed', async () => {
  const { errors } = await actions
    .createPostNumberNotNullPermission({ views: null })

  var authorizationFailed = hasAuthorizationError(errors)
  expect.equal(authorizationFailed, true)
})

test('number null permission failed', async () => {
  const { errors } = await actions
    .createPostNumberNullPermission({})

  var authorizationFailed = hasAuthorizationError(errors)
  expect.equal(authorizationFailed, true)
})

test('boolean not null permission successful', async () => {
  const { errors } = await actions
    .createPostBooleanNotNullPermission({ active: true })

  var authorizationFailed = hasAuthorizationError(errors)
  expect.equal(authorizationFailed, false)
})

test('boolean not null permission failed', async () => {
  const { errors } = await actions
    .createPostBooleanNotNullPermission({ active: null })

  var authorizationFailed = hasAuthorizationError(errors)
  expect.equal(authorizationFailed, true)
})

test('boolean null permission failed', async () => {
  const { errors } = await actions
    .createPostBooleanNullPermission({})

  var authorizationFailed = hasAuthorizationError(errors)
  expect.equal(authorizationFailed, true)
})

function hasAuthorizationError(errors?): boolean {
  if (errors == null)
    return false;

  var hasError = false
   errors.forEach(function(error) {
    if(error.message == 'not authorized to access this operation') {
      hasError = true
    }
  });
  
  return hasError;
}