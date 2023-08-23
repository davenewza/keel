import { models, actions, resetDatabase } from "@teamkeel/testing";
import { test, expect, beforeEach } from "vitest";

beforeEach(resetDatabase);

test("user has authorisation to read their own record", async () => {
  const identity = await models.identity.create({
    email: "adam@keel.xyz",
    password: "foo",
  });
  const user = await models.user.create({
    name: "Adam",
    identityId: identity.id,
  });

  await expect(
    actions.withIdentity(identity).getUser({
      id: user.id,
    })
  ).not.toHaveAuthorizationError();
});

test("user does not have authorisation to read another user's record", async () => {
  const identityAdam = await models.identity.create({
    email: "adam@keel.xyz",
    password: "foo",
  });
  const userAdam = await models.user.create({
    name: "Adam",
    identityId: identityAdam.id,
  });

  const identityDave = await models.identity.create({
    email: "dave@keel.xyz",
    password: "foo",
  });
  const userDave = await models.user.create({
    name: "Dave",
    identityId: identityDave.id,
  });

  await expect(
    actions.withIdentity(identityDave).getUser({
      id: userAdam.id,
    })
  ).toHaveAuthorizationError();
});

test("user in a shared organisation has authorisation to read other user's records from the organisation", async () => {
  const organisationAdam = await models.organisation.create({
    name: "AdamCo",
  });
  const organisationDave = await models.organisation.create({
    name: "DaveCo",
  });
  const organisationKeel = await models.organisation.create({
    name: "Keel",
  });

  const identityAdam = await models.identity.create({
    email: "adam@keel.xyz",
    password: "foo",
  });
  const userAdam = await models.user.create({
    name: "Adam",
    identityId: identityAdam.id,
  });

  const identityDave = await models.identity.create({
    email: "dave@keel.xyz",
    password: "foo",
  });
  const userDave = await models.user.create({
    name: "Dave",
    identityId: identityDave.id,
  });

  await models.userOrganisation.create({
    organisationId: organisationAdam.id,
    userId: userAdam.id,
  });
  await models.userOrganisation.create({
    organisationId: organisationDave.id,
    userId: userDave.id,
  });
  await models.userOrganisation.create({
    organisationId: organisationKeel.id,
    userId: userDave.id,
  });
  await models.userOrganisation.create({
    organisationId: organisationKeel.id,
    userId: userAdam.id,
  });

  await expect(
    actions.withIdentity(identityDave).getUser({
      id: userAdam.id,
    })
  ).not.toHaveAuthorizationError();
});

test("user does not have authorisation to read user's records from another organisation", async () => {
  const organisationAdam = await models.organisation.create({
    name: "AdamCo",
  });
  const organisationDave = await models.organisation.create({
    name: "DaveCo",
  });

  const identityAdam = await models.identity.create({
    email: "adam@keel.xyz",
    password: "foo",
  });
  const userAdam = await models.user.create({
    name: "Adam",
    identityId: identityAdam.id,
  });

  const identityDave = await models.identity.create({
    email: "dave@keel.xyz",
    password: "foo",
  });
  const userDave = await models.user.create({
    name: "Dave",
    identityId: identityDave.id,
  });

  await models.userOrganisation.create({
    organisationId: organisationAdam.id,
    userId: userAdam.id,
  });
  await models.userOrganisation.create({
    organisationId: organisationDave.id,
    userId: userDave.id,
  });

  await expect(
    actions.withIdentity(identityDave).getUser({
      id: userAdam.id,
    })
  ).toHaveAuthorizationError();
});

test("user has authorisation to list records from own organisation", async () => {
  const organisationAdam = await models.organisation.create({
    name: "AdamCo",
  });
  const organisationDave = await models.organisation.create({
    name: "DaveCo",
  });
  const organisationKeel = await models.organisation.create({
    name: "Keel",
  });

  const identityAdam = await models.identity.create({
    email: "adam@keel.xyz",
    password: "foo",
  });
  const userAdam = await models.user.create({
    name: "Adam",
    identityId: identityAdam.id,
  });

  const identityDave = await models.identity.create({
    email: "dave@keel.xyz",
    password: "foo",
  });
  const userDave = await models.user.create({
    name: "Dave",
    identityId: identityDave.id,
  });

  await models.userOrganisation.create({
    organisationId: organisationAdam.id,
    userId: userAdam.id,
  });
  await models.userOrganisation.create({
    organisationId: organisationDave.id,
    userId: userDave.id,
  });
  await models.userOrganisation.create({
    organisationId: organisationKeel.id,
    userId: userDave.id,
  });
  await models.userOrganisation.create({
    organisationId: organisationKeel.id,
    userId: userAdam.id,
  });

  const results = await actions
    .withIdentity(identityDave)
    .listUsersByOrganisation({
      where: {
        organisations: {
          organisation: { id: { equals: organisationDave.id } },
        },
      },
    });

  expect(results.results).toHaveLength(1);
});

test("user does not have authorisation to list records from another organisation", async () => {
  const organisationAdam = await models.organisation.create({
    name: "AdamCo",
  });
  const organisationDave = await models.organisation.create({
    name: "DaveCo",
  });
  const organisationKeel = await models.organisation.create({
    name: "Keel",
  });

  const identityAdam = await models.identity.create({
    email: "adam@keel.xyz",
    password: "foo",
  });
  const userAdam = await models.user.create({
    name: "Adam",
    identityId: identityAdam.id,
  });

  const identityDave = await models.identity.create({
    email: "dave@keel.xyz",
    password: "foo",
  });
  const userDave = await models.user.create({
    name: "Dave",
    identityId: identityDave.id,
  });

  await models.userOrganisation.create({
    organisationId: organisationAdam.id,
    userId: userAdam.id,
  });
  await models.userOrganisation.create({
    organisationId: organisationDave.id,
    userId: userDave.id,
  });
  await models.userOrganisation.create({
    organisationId: organisationKeel.id,
    userId: userDave.id,
  });
  await models.userOrganisation.create({
    organisationId: organisationKeel.id,
    userId: userAdam.id,
  });

  await expect(
    actions.withIdentity(identityDave).listUsersByOrganisation({
      where: {
        organisations: {
          organisation: { id: { equals: organisationAdam.id } },
        },
      },
    })
  ).toHaveAuthorizationError();
});

test("not authorised to create organisation with no identity", async () => {
  const identityAdam = await models.identity.create({
    email: "adam@keel.xyz",
    password: "foo",
  });
  const userAdam = await models.user.create({
    name: "Adam",
    identityId: identityAdam.id,
  });

  const identityDave = await models.identity.create({
    email: "dave@keel.xyz",
    password: "foo",
  });
  const userDave = await models.user.create({
    name: "Dave",
    identityId: identityDave.id,
  });

  await expect(
    actions.createOrganisation({
      name: "Keel",
      users: [{ user: { id: userAdam.id } }, { user: { id: userDave.id } }],
    })
  ).toHaveAuthorizationError();
});

test("authorised to create organisation when identity belongs to organisation", async () => {
  const identityAdam = await models.identity.create({
    email: "adam@keel.xyz",
    password: "foo",
  });
  const userAdam = await models.user.create({
    name: "Adam",
    identityId: identityAdam.id,
  });

  const identityDave = await models.identity.create({
    email: "dave@keel.xyz",
    password: "foo",
  });
  const userDave = await models.user.create({
    name: "Dave",
    identityId: identityDave.id,
  });

  const organisation = await actions
    .withIdentity(identityDave)
    .createOrganisation({
      name: "Keel",
      users: [{ user: { id: userAdam.id } }, { user: { id: userDave.id } }],
    });
});

test("not authorised to create organisation when identity does not belong to organisation", async () => {
  const identityAdam = await models.identity.create({
    email: "adam@keel.xyz",
    password: "foo",
  });
  const userAdam = await models.user.create({
    name: "Adam",
    identityId: identityAdam.id,
  });

  const identityDave = await models.identity.create({
    email: "dave@keel.xyz",
    password: "foo",
  });
  const userDave = await models.user.create({
    name: "Dave",
    identityId: identityDave.id,
  });

  await expect(
    actions.withIdentity(identityDave).createOrganisation({
      name: "Adam Co",
      users: [{ user: { id: userAdam.id } }],
    })
  ).toHaveAuthorizationError();
});

test("only list organisations which identity is associated with", async () => {
  const organisationAdam = await models.organisation.create({
    name: "AdamCo",
  });
  const organisationDave = await models.organisation.create({
    name: "DaveCo",
  });
  const organisationKeel = await models.organisation.create({
    name: "Keel",
  });

  const identityAdam = await models.identity.create({
    email: "adam@keel.xyz",
    password: "foo",
  });
  const userAdam = await models.user.create({
    name: "Adam",
    identityId: identityAdam.id,
  });

  const identityDave = await models.identity.create({
    email: "dave@keel.xyz",
    password: "foo",
  });
  const userDave = await models.user.create({
    name: "Dave",
    identityId: identityDave.id,
  });

  await models.userOrganisation.create({
    organisationId: organisationAdam.id,
    userId: userAdam.id,
  });
  await models.userOrganisation.create({
    organisationId: organisationDave.id,
    userId: userDave.id,
  });
  await models.userOrganisation.create({
    organisationId: organisationKeel.id,
    userId: userDave.id,
  });
  await models.userOrganisation.create({
    organisationId: organisationKeel.id,
    userId: userAdam.id,
  });

  const rows = await actions.withIdentity(identityDave).listOrganisations({});
  expect(rows.results).toHaveLength(2);
  expect(rows.results[0].id).not.toEqual(organisationAdam.id);
  expect(rows.results[1].id).not.toEqual(organisationAdam.id);
});

test("list no organisations when there is no identity", async () => {
  const organisationAdam = await models.organisation.create({
    name: "AdamCo",
  });
  const organisationDave = await models.organisation.create({
    name: "DaveCo",
  });
  const organisationKeel = await models.organisation.create({
    name: "Keel",
  });

  const identityAdam = await models.identity.create({
    email: "adam@keel.xyz",
    password: "foo",
  });
  const userAdam = await models.user.create({
    name: "Adam",
    identityId: identityAdam.id,
  });

  const identityDave = await models.identity.create({
    email: "dave@keel.xyz",
    password: "foo",
  });
  const userDave = await models.user.create({
    name: "Dave",
    identityId: identityDave.id,
  });

  await models.userOrganisation.create({
    organisationId: organisationAdam.id,
    userId: userAdam.id,
  });
  await models.userOrganisation.create({
    organisationId: organisationDave.id,
    userId: userDave.id,
  });
  await models.userOrganisation.create({
    organisationId: organisationKeel.id,
    userId: userDave.id,
  });
  await models.userOrganisation.create({
    organisationId: organisationKeel.id,
    userId: userAdam.id,
  });

  const rows = await actions.listOrganisations({});
  expect(rows.results).toHaveLength(0);
});

test("authorised to get an organisations which the identity is associated to", async () => {
  const organisationAdam = await models.organisation.create({
    name: "AdamCo",
  });
  const organisationDave = await models.organisation.create({
    name: "DaveCo",
  });
  const organisationKeel = await models.organisation.create({
    name: "Keel",
  });

  const identityAdam = await models.identity.create({
    email: "adam@keel.xyz",
    password: "foo",
  });
  const userAdam = await models.user.create({
    name: "Adam",
    identityId: identityAdam.id,
  });

  const identityDave = await models.identity.create({
    email: "dave@keel.xyz",
    password: "foo",
  });
  const userDave = await models.user.create({
    name: "Dave",
    identityId: identityDave.id,
  });

  await models.userOrganisation.create({
    organisationId: organisationAdam.id,
    userId: userAdam.id,
  });
  await models.userOrganisation.create({
    organisationId: organisationDave.id,
    userId: userDave.id,
  });
  await models.userOrganisation.create({
    organisationId: organisationKeel.id,
    userId: userDave.id,
  });
  await models.userOrganisation.create({
    organisationId: organisationKeel.id,
    userId: userAdam.id,
  });

  const row = await actions
    .withIdentity(identityDave)
    .getOrganisation({ id: organisationKeel.id });
  expect(row!.id).toEqual(organisationKeel.id);
});

test("not authorised to get an organisations which the identity is not associated to", async () => {
  const organisationAdam = await models.organisation.create({
    name: "AdamCo",
  });
  const organisationDave = await models.organisation.create({
    name: "DaveCo",
  });
  const organisationKeel = await models.organisation.create({
    name: "Keel",
  });

  const identityAdam = await models.identity.create({
    email: "adam@keel.xyz",
    password: "foo",
  });
  const userAdam = await models.user.create({
    name: "Adam",
    identityId: identityAdam.id,
  });

  const identityDave = await models.identity.create({
    email: "dave@keel.xyz",
    password: "foo",
  });
  const userDave = await models.user.create({
    name: "Dave",
    identityId: identityDave.id,
  });

  await models.userOrganisation.create({
    organisationId: organisationAdam.id,
    userId: userAdam.id,
  });
  await models.userOrganisation.create({
    organisationId: organisationDave.id,
    userId: userDave.id,
  });
  await models.userOrganisation.create({
    organisationId: organisationKeel.id,
    userId: userDave.id,
  });
  await models.userOrganisation.create({
    organisationId: organisationKeel.id,
    userId: userAdam.id,
  });

  await expect(
    actions
      .withIdentity(identityDave)
      .getOrganisation({ id: organisationAdam.id })
  ).toHaveAuthorizationError();
});

test("not authorised to get an organisations with no identity", async () => {
  const organisationAdam = await models.organisation.create({
    name: "AdamCo",
  });
  const organisationDave = await models.organisation.create({
    name: "DaveCo",
  });
  const organisationKeel = await models.organisation.create({
    name: "Keel",
  });

  const identityAdam = await models.identity.create({
    email: "adam@keel.xyz",
    password: "foo",
  });
  const userAdam = await models.user.create({
    name: "Adam",
    identityId: identityAdam.id,
  });

  const identityDave = await models.identity.create({
    email: "dave@keel.xyz",
    password: "foo",
  });
  const userDave = await models.user.create({
    name: "Dave",
    identityId: identityDave.id,
  });

  await models.userOrganisation.create({
    organisationId: organisationAdam.id,
    userId: userAdam.id,
  });
  await models.userOrganisation.create({
    organisationId: organisationDave.id,
    userId: userDave.id,
  });
  await models.userOrganisation.create({
    organisationId: organisationKeel.id,
    userId: userDave.id,
  });
  await models.userOrganisation.create({
    organisationId: organisationKeel.id,
    userId: userAdam.id,
  });

  await expect(
    actions.getOrganisation({ id: organisationAdam.id })
  ).toHaveAuthorizationError();
});