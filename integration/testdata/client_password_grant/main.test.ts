import { actions, models, resetDatabase } from "@teamkeel/testing";
import { test, expect, beforeEach, beforeAll } from "vitest";
import { APIClient, TokenStore } from "./keelClient";

const baseUrl = process.env.KEEL_TESTING_CLIENT_API_URL!;

beforeEach(resetDatabase);

test("authenticateWithPassword - with default token stores", async () => {
  const client = new APIClient({ baseUrl });

  await client.auth.authenticateWithPassword("user@example.com", "1234");
  expect(await client.auth.isAuthenticated()).toBeTruthy();
  expect(await client.auth.expiresAt()).not.toBeNull();
  expect((await client.auth.expiresAt()!) > new Date()).toBeTruthy();
  expect(client.auth.accessToken.get()).not.toBeNull();
  expect(client.auth.refreshToken.get()).not.toBeNull();

  await client.auth.authenticateWithPassword("user@example.com", "oops");
  expect(await client.auth.isAuthenticated()).not.toBeTruthy();
  expect(await client.auth.expiresAt()).toBeNull();
  expect(client.auth.accessToken.get()).toBeNull();
  expect(client.auth.refreshToken.get()).toBeNull();

  await models.post.create({ title: "Test" });
  const response1 = await client.api.queries.allPosts();
  expect(response1.error!.type).toEqual("forbidden");

  await client.auth.authenticateWithPassword("user@example.com", "1234");
  expect(await client.auth.isAuthenticated()).toBeTruthy();
  expect(await client.auth.expiresAt()).not.toBeNull();
  expect((await client.auth.expiresAt()!) > new Date()).toBeTruthy();
  expect(client.auth.accessToken.get()).not.toBeNull();
  expect(client.auth.refreshToken.get()).not.toBeNull();

  const response2 = await client.api.queries.allPosts();
  expect(response2.data?.results).toHaveLength(1);
});

test("authenticateWithPassword - with custom token stores", async () => {
  const accessTokenStore = new TestTokenStore();
  const refreshTokenStore = new TestTokenStore();

  const client = new APIClient({
    baseUrl,
    accessToken: accessTokenStore,
    refreshToken: refreshTokenStore,
  });

  await client.auth.authenticateWithPassword("user@example.com", "1234");
  expect(await client.auth.isAuthenticated()).toBeTruthy();
  expect(await client.auth.expiresAt()).not.toBeNull();
  expect((await client.auth.expiresAt()!) > new Date()).toBeTruthy();
  expect(accessTokenStore.get()).not.toBeNull();
  expect(refreshTokenStore.get()).not.toBeNull();
  expect(client.auth.accessToken.get()).not.toBeNull();
  expect(client.auth.refreshToken.get()).not.toBeNull();
  expect(accessTokenStore.get()).toEqual(client.auth.accessToken.get());
  expect(refreshTokenStore.get()).toEqual(client.auth.refreshToken.get());

  await client.auth.authenticateWithPassword("user@example.com", "oops");
  expect(await client.auth.isAuthenticated()).not.toBeTruthy();
  expect(await client.auth.expiresAt()).toBeNull();
  expect(accessTokenStore.get()).toBeNull();
  expect(refreshTokenStore.get()).toBeNull();
  expect(client.auth.accessToken.get()).toBeNull();
  expect(client.auth.refreshToken.get()).toBeNull();
  expect(accessTokenStore.get()).toEqual(client.auth.accessToken.get());
  expect(refreshTokenStore.get()).toEqual(client.auth.refreshToken.get());

  await models.post.create({ title: "Test" });
  const response1 = await client.api.queries.allPosts();
  expect(response1.error!.type).toEqual("forbidden");

  await client.auth.authenticateWithPassword("user@example.com", "1234");
  expect(await client.auth.isAuthenticated()).toBeTruthy();
  expect(await client.auth.expiresAt()).not.toBeNull();
  expect((await client.auth.expiresAt()!) > new Date()).toBeTruthy();
  expect(accessTokenStore.get()).not.toBeNull();
  expect(refreshTokenStore.get()).not.toBeNull();
  expect(client.auth.accessToken.get()).not.toBeNull();
  expect(client.auth.refreshToken.get()).not.toBeNull();
  expect(accessTokenStore.get()).toEqual(client.auth.accessToken.get());
  expect(refreshTokenStore.get()).toEqual(client.auth.refreshToken.get());

  const response2 = await client.api.queries.allPosts();
  expect(response2.data?.results).toHaveLength(1);
});

test("valid access token", async () => {
  const accessTokenStore = new TestTokenStore();

  const client = new APIClient({
    baseUrl,
    accessToken: accessTokenStore,
  });

  await client.auth.authenticateWithPassword("user@example.com", "1234");
  expect(await client.auth.isAuthenticated()).toBeTruthy();

  expect(accessTokenStore.get()).not.toBeNull();
  expect(accessTokenStore.get()).not.toEqual("");
  expect(accessTokenStore.get()).toEqual(client.auth.accessToken.get());

  client.auth.accessToken.set("1234");
  expect(accessTokenStore.get()).toEqual("1234");

  client.auth.accessToken.set(null);
  expect(accessTokenStore.get()).toBeNull();
});

test("valid refresh token", async () => {
  const refreshTokenStore = new TestTokenStore();

  const client = new APIClient({
    baseUrl,
    refreshToken: refreshTokenStore,
  });

  await client.auth.authenticateWithPassword("user@example.com", "1234");
  expect(await client.auth.isAuthenticated()).toBeTruthy();

  expect(refreshTokenStore.get()).not.toBeNull();
  expect(refreshTokenStore.get()).not.toEqual("");
  expect(refreshTokenStore.get()).toEqual(client.auth.refreshToken.get());

  client.auth.refreshToken.set("1234");
  expect(refreshTokenStore.get()).toEqual("1234");

  client.auth.refreshToken.set(null);
  expect(refreshTokenStore.get()).toBeNull();
});

test("refreshing successfully", async () => {
  const client = new APIClient({ baseUrl });

  await client.auth.authenticateWithPassword("user@example.com", "1234");
  expect(await client.auth.isAuthenticated()).toBeTruthy();

  const accessToken = client.auth.accessToken.get();
  const refreshToken = client.auth.refreshToken.get();

  expect(accessToken).not.toBeNull();
  expect(refreshToken).not.toBeNull();

  const expiry1 = client.auth.expiresAt();
  expect(expiry1).not.toBeNull();

  await delay(1000); // 1000ms is the smallest increment we can
  const refreshed = await client.auth.refresh();
  expect(refreshed).toBeTruthy();

  expect(client.auth.accessToken.get()).not.toBeNull();
  expect(client.auth.refreshToken.get()).not.toBeNull();
  expect(client.auth.accessToken.get()).not.toEqual(accessToken);
  expect(client.auth.refreshToken.get()).not.toEqual(refreshToken);

  const expiry2 = client.auth.expiresAt();
  expect(expiry1?.getTime()).lessThan(expiry2!.getTime());
});

test("logout successfully", async () => {
  const client = new APIClient({ baseUrl });

  await client.auth.authenticateWithPassword("user@example.com", "1234");
  expect(await client.auth.isAuthenticated()).toBeTruthy();

  await client.auth.logout();
  expect(client.auth.accessToken.get()).toBeNull();
  expect(client.auth.refreshToken.get()).toBeNull();

  expect(await client.auth.expiresAt()).toBeNull();

  expect(await client.auth.isAuthenticated()).not.toBeTruthy();
  expect(client.auth.accessToken.get()).toBeNull();
  expect(client.auth.refreshToken.get()).toBeNull();

  expect(await client.auth.refresh()).not.toBeTruthy();
  expect(client.auth.accessToken.get()).toBeNull();
  expect(client.auth.refreshToken.get()).toBeNull();
});

test("logout revokes refresh token on server successfully", async () => {
  const client = new APIClient({ baseUrl });

  await client.auth.authenticateWithPassword("user@example.com", "1234");
  expect(await client.auth.isAuthenticated()).toBeTruthy();

  await client.auth.logout();

  const accessToken = client.auth.accessToken.get();
  const refreshToken = client.auth.refreshToken.get();

  expect(client.auth.accessToken.get()).toBeNull();
  expect(client.auth.refreshToken.get()).toBeNull();

  expect(await client.auth.expiresAt()).toBeNull();
  expect(await client.auth.isAuthenticated()).not.toBeTruthy();

  client.auth.accessToken.set(accessToken);
  client.auth.refreshToken.set(refreshToken);

  const refresh = await client.auth.refresh();
  expect(refresh).not.toBeTruthy();
  expect(await client.auth.isAuthenticated()).not.toBeTruthy();

  expect(client.auth.accessToken.get()).toBeNull();
  expect(client.auth.refreshToken.get()).toBeNull();
});

test("authentication flow with default token store", async () => {
  const client = new APIClient({ baseUrl });

  await client.auth.authenticateWithPassword("user@example.com", "1234");
  expect(await client.auth.isAuthenticated()).toBeTruthy();
  expect(await client.auth.expiresAt()).not.toBeNull();
  expect((await client.auth.expiresAt()!) > new Date()).toBeTruthy();

  await client.auth.authenticateWithPassword("user@example.com", "oops");
  expect(await client.auth.isAuthenticated()).not.toBeTruthy();
  expect(await client.auth.expiresAt()).toBeNull();

  await client.auth.authenticateWithPassword("user@example.com", "1234");
  expect(await client.auth.isAuthenticated()).toBeTruthy();
  expect(await client.auth.expiresAt()).not.toBeNull();
  expect((await client.auth.expiresAt()!) > new Date()).toBeTruthy();

  const expiry1 = client.auth.expiresAt();

  await delay(1000);
  const refreshed = await client.auth.refresh();
  expect(refreshed).toBeTruthy();
  expect(await client.auth.isAuthenticated()).toBeTruthy();

  const expiry2 = client.auth.expiresAt();
  expect(expiry1?.getTime()).lessThan(expiry2!.getTime());

  await client.auth.logout();
  expect(await client.auth.isAuthenticated()).not.toBeTruthy();

  const refreshed2 = await client.auth.refresh();
  expect(refreshed2).not.toBeTruthy();
  expect(await client.auth.isAuthenticated()).not.toBeTruthy();
});

test("volatile access token - refreshes correctly on isAuthenticated()", async () => {
  const refreshTokenStore = new TestTokenStore();

  let client = new APIClient({
    baseUrl,
    refreshToken: refreshTokenStore,
  });

  await client.auth.authenticateWithPassword("user@example.com", "1234");
  expect(await client.auth.isAuthenticated()).toBeTruthy();
  expect(client.auth.accessToken.get()).not.toBeNull();
  expect(client.auth.refreshToken.get()).not.toBeNull();

  client = new APIClient({
    baseUrl,
    refreshToken: refreshTokenStore,
  });
  expect(client.auth.accessToken.get()).toBeNull();
  expect(client.auth.refreshToken.get()).not.toBeNull();

  expect(await client.auth.isAuthenticated()).toBeTruthy();
  expect(client.auth.accessToken.get()).not.toBeNull();
  expect(client.auth.refreshToken.get()).not.toBeNull();
});

test("volatile access token - refreshes correctly refresh()", async () => {
  const refreshTokenStore = new TestTokenStore();

  let client = new APIClient({
    baseUrl,
    refreshToken: refreshTokenStore,
  });

  await client.auth.authenticateWithPassword("user@example.com", "1234");
  expect(await client.auth.isAuthenticated()).toBeTruthy();
  expect(client.auth.accessToken.get()).not.toBeNull();
  expect(client.auth.refreshToken.get()).not.toBeNull();

  client = new APIClient({
    baseUrl,
    refreshToken: refreshTokenStore,
  });
  expect(client.auth.accessToken.get()).toBeNull();
  expect(client.auth.refreshToken.get()).not.toBeNull();

  expect(await client.auth.refresh()).toBeTruthy();
  expect(client.auth.accessToken.get()).not.toBeNull();
  expect(client.auth.refreshToken.get()).not.toBeNull();
});

test("volatile access token - refreshes correctly refresh()", async () => {
  const refreshTokenStore = new TestTokenStore();

  let client = new APIClient({
    baseUrl,
    refreshToken: refreshTokenStore,
  });

  await client.auth.authenticateWithPassword("user@example.com", "1234");
  expect(await client.auth.isAuthenticated()).toBeTruthy();
  expect(client.auth.accessToken.get()).not.toBeNull();
  expect(client.auth.refreshToken.get()).not.toBeNull();

  client = new APIClient({
    baseUrl,
    refreshToken: refreshTokenStore,
  });
  expect(client.auth.accessToken.get()).toBeNull();
  expect(client.auth.refreshToken.get()).not.toBeNull();

  expect(await client.auth.refresh()).toBeTruthy();
  expect(client.auth.accessToken.get()).not.toBeNull();
  expect(client.auth.refreshToken.get()).not.toBeNull();
});

test("access token expiring and refreshing until refresh token expires", async () => {
  let client = new APIClient({
    baseUrl,
  });

  await client.auth.authenticateWithPassword("user@example.com", "1234");
  expect(await client.auth.isAuthenticated()).toBeTruthy();
  expect(client.auth.accessToken.get()).not.toBeNull();
  expect(client.auth.refreshToken.get()).not.toBeNull();

  let currentAccessToken = client.auth.accessToken.get();
  let currentRefreshToken = client.auth.refreshToken.get();

  let response = await client.api.queries.allPosts();
  expect(response.error).toBeUndefined();
  expect(client.auth.accessToken.get()).toEqual(currentAccessToken);
  expect(client.auth.refreshToken.get()).toEqual(currentRefreshToken);

  await delay(25);

  response = await client.api.queries.allPosts();
  expect(response.error).toBeUndefined();
  expect(client.auth.accessToken.get()).toEqual(currentAccessToken);
  expect(client.auth.refreshToken.get()).toEqual(currentRefreshToken);

  await delay(1000);

  response = await client.api.queries.allPosts();
  expect(response.error).toBeUndefined();
  expect(client.auth.accessToken.get()).not.toEqual(currentAccessToken);
  expect(client.auth.refreshToken.get()).not.toEqual(currentRefreshToken);
  currentAccessToken = client.auth.accessToken.get();
  currentRefreshToken = client.auth.refreshToken.get();

  await delay(25);

  response = await client.api.queries.allPosts();
  expect(response.error).toBeUndefined();
  expect(client.auth.accessToken.get()).toEqual(currentAccessToken);
  expect(client.auth.refreshToken.get()).toEqual(currentRefreshToken);

  await delay(1000);

  response = await client.api.queries.allPosts();
  expect(response.error).toBeUndefined();
  expect(client.auth.accessToken.get()).not.toEqual(currentAccessToken);
  expect(client.auth.refreshToken.get()).not.toEqual(currentRefreshToken);
  currentAccessToken = client.auth.accessToken.get();
  currentRefreshToken = client.auth.refreshToken.get();

  await delay(2000);

  response = await client.api.queries.allPosts();
  expect(response.error!.type).toEqual("forbidden");
  expect(client.auth.accessToken.get()).toBeNull();
  expect(client.auth.refreshToken.get()).toBeNull();
});

class TestTokenStore implements TokenStore {
  private token: string | null = null;

  public constructor() {}

  get = () => {
    return this.token;
  };

  set = (token: string | null): void => {
    this.token = token;
  };
}

function delay(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
