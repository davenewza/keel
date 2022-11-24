import { test, expect, actions, Thing, logger } from "@teamkeel/testing";
import { LogLevel } from "@teamkeel/sdk";

test("eqAndNotEq", async () => {
  await Thing.create({ switchIsOn: null });
  await Thing.create({ switchIsOn: false });
  await Thing.create({ switchIsOn: true });

  let resp;

  resp = await actions.eqArg({ where: { arg: null } });
  expect(resp.collection.map((thing) => thing.switchIsOn)).toEqual([null]);

  resp = await actions.eqArg({ where: { arg: false } });
  expect(resp.collection.map((thing) => thing.switchIsOn)).toEqual([false]);

  resp = await actions.eqArg({ where: { arg: true } });
  expect(resp.collection.map((thing) => thing.switchIsOn)).toEqual([true]);

  let nullsLast = function (a, b) {
    if (a === null) {
      return 1;
    }
    if (b === null) {
      return -1;
    }
    return a < b ? -1 : 1;
  };

  resp = await actions.notEqArg({ where: { arg: null } });
  expect(
    resp.collection.map((thing) => thing.switchIsOn).sort(nullsLast)
  ).toEqual([false, true]);

  resp = await actions.notEqArg({ where: { arg: false } });
  expect(
    resp.collection.map((thing) => thing.switchIsOn).sort(nullsLast)
  ).toEqual([true, null]);

  resp = await actions.notEqArg({ where: { arg: true } });
  expect(
    resp.collection.map((thing) => thing.switchIsOn).sort(nullsLast)
  ).toEqual([false, null]);
});