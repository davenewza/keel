import { models, jobs, resetDatabase } from "@teamkeel/testing";
import { test, expect, beforeEach } from "vitest";

beforeEach(resetDatabase);

async function jobRan(id) {
  const track = await models.trackJob.findOne({ id });
  return track!.didJobRun;
}

test("job - without identity - not permitted", async () => {
  const { id } = await models.trackJob.create({ didJobRun: false });

  await expect(jobs.manualJob({ id })).toHaveAuthorizationError();

  expect(await jobRan(id)).toBeFalsy();

  await expect(jobs.manualJobMultiRoles({ id })).toHaveAuthorizationError();

  expect(await jobRan(id)).toBeFalsy();
});

test("job - without identity, true expression permission -  permitted", async () => {
  const { id } = await models.trackJob.create({ didJobRun: false });

  await expect(
    jobs.manualJobTrueExpression({ id })
  ).not.toHaveAuthorizationError();

  expect(await jobRan(id)).toBeTruthy();
});

test("job - wrong domain - not permitted", async () => {
  const { id } = await models.trackJob.create({ didJobRun: false });
  const identity = await models.identity.create({ email: "weave@gmail.com" });

  await expect(
    jobs.withIdentity(identity).manualJob({ id })
  ).toHaveAuthorizationError();

  expect(await jobRan(id)).toBeFalsy();

  await expect(
    jobs.withIdentity(identity).manualJobMultiRoles({ id })
  ).toHaveAuthorizationError();

  expect(await jobRan(id)).toBeFalsy();
});

test("job - authorised domain - permitted", async () => {
  const { id } = await models.trackJob.create({ didJobRun: false });
  const identity = await models.identity.create({ email: "keel@keel.so" });

  await expect(
    jobs.withIdentity(identity).manualJob({ id })
  ).not.toHaveAuthorizationError();

  expect(await jobRan(id)).toBeTruthy();
});

test("job - wrong authorised domain - not permitted", async () => {
  const { id } = await models.trackJob.create({ didJobRun: false });
  const identity = await models.identity.create({ email: "keel@keel.dev" });

  await expect(
    jobs.withIdentity(identity).manualJob({ id })
  ).toHaveAuthorizationError();

  expect(await jobRan(id)).toBeFalsy();
});

test("job - multi domains, authorised domain - permitted", async () => {
  const { id } = await models.trackJob.create({ didJobRun: false });
  const identity = await models.identity.create({ email: "keel@keel.so" });

  await expect(
    jobs.withIdentity(identity).manualJobMultiRoles({ id })
  ).not.toHaveAuthorizationError();

  expect(await jobRan(id)).toBeTruthy();
});

test("job - true expression - permitted", async () => {
  const { id } = await models.trackJob.create({ didJobRun: false });

  await expect(
    jobs.manualJobTrueExpression({ id })
  ).not.toHaveAuthorizationError();

  expect(await jobRan(id)).toBeTruthy();
});

test("job - env var expression - not permitted", async () => {
  const { id } = await models.trackJob.create({ didJobRun: false });

  await expect(
    jobs.manualJobEnvExpression({ id })
  ).not.toHaveAuthorizationError();

  expect(await jobRan(id)).toBeTruthy();
});

test("job - multiple permissions - not permitted", async () => {
  const { id } = await models.trackJob.create({ didJobRun: false });
  const identity = await models.identity.create({ email: "bob@bob.com" });

  await expect(
    jobs.withIdentity(identity).manualJobMultiPermission({ id })
  ).toHaveAuthorizationError();

  expect(await jobRan(id)).toBeFalsy();
});

test("job - multiple permissions - permitted", async () => {
  const { id } = await models.trackJob.create({ didJobRun: false });
  const identity = await models.identity.create({ email: "keelson@keel.so" });

  await expect(
    jobs.withIdentity(identity).manualJobMultiPermission({ id })
  ).not.toHaveAuthorizationError();

  expect(await jobRan(id)).toBeTruthy();
});

test("job - allowed in job code - permitted", async () => {
  const { id } = await models.trackJob.create({ didJobRun: false });
  const identity = await models.identity.create({ email: "keel@keel.so" });

  await expect(
    jobs.withIdentity(identity).manualJobDeniedInCode({ id, denyIt: false })
  ).not.toHaveAuthorizationError();

  expect(await jobRan(id)).toBeTruthy();
});

test("job - denied in job code - not permitted without rollback transaction", async () => {
  const { id } = await models.trackJob.create({ didJobRun: false });
  const identity = await models.identity.create({ email: "keel@keel.so" });

  await expect(
    jobs.withIdentity(identity).manualJobDeniedInCode({ id, denyIt: true })
  ).toHaveAuthorizationError();

  // This would be false if a transaction rolled back.
  expect(await jobRan(id)).toBeTruthy();
});

test("job - exception - internal error without rollback transaction", async () => {
  const { id } = await models.trackJob.create({ didJobRun: false });
  const identity = await models.identity.create({ email: "keel@keel.so" });

  await expect(
    jobs.withIdentity(identity).manualJobWithException({ id })
  ).toHaveError({
    code: "ERR_INTERNAL",
  });

  // This would be false if a transaction rolled back.
  expect(await jobRan(id)).toBeTruthy();
});

test("job - scheduled without permission", async () => {
  await expect(
    jobs.scheduledWithoutPermissions({ scheduled: true })
  ).not.toHaveAuthorizationError();
});