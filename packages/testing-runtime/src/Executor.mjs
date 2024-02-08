import jwt from "jsonwebtoken";

export class Executor {
  constructor(props) {
    this._identity = props.identity || null;
    this._authToken = props.authToken || null;
    this._apiBaseUrl = props.apiBaseUrl;
    this._parseJsonResult = props.parseJsonResult;

    // Return a proxy which will return a bound version of the
    // _execute method for any unknown properties. This creates
    // the actions API we want but in a dynamic way without needing
    // codegen. We then generate the right type definitions for
    // this class in the @teamkeel/testing package.
    return new Proxy(this, {
      get(target, prop) {
        const v = Reflect.get(...arguments);
        if (v !== undefined) {
          return v;
        }
        return target._execute.bind(target, prop);
      },
    });
  }
  withIdentity(i) {
    return new Executor({
      identity: i,
      apiBaseUrl: this._apiBaseUrl,
      parseJsonResult: this._parseJsonResult,
    });
  }
  withAuthToken(t) {
    return new Executor({
      authToken: t,
      apiBaseUrl: this._apiBaseUrl,
      parseJsonResult: this._parseJsonResult,
    });
  }
  _execute(method, params) {
    const headers = { "Content-Type": "application/json" };

    // An Identity instance is provided make a JWT
    if (this._identity !== null) {
      const base64pk = process.env.KEEL_DEFAULT_PK;
      let privateKey = undefined;

      if (base64pk) {
        privateKey = Buffer.from(base64pk, "base64").toString("utf8");
      }

      headers["Authorization"] =
        "Bearer " +
        jwt.sign({}, privateKey, {
          algorithm: privateKey ? "RS256" : "none",
          expiresIn: 60 * 60 * 24,
          subject: this._identity.id,
          issuer: "https://keel.so",
        });
    }

    // If an auth token is provided that can be sent as-is
    if (this._authToken !== null) {
      headers["Authorization"] = "Bearer " + this._authToken;
    }

    if (params?.scheduled) {
      headers["X-Trigger-Type"] = "scheduled";
    } else {
      headers["X-Trigger-Type"] = "manual";
    }

    // Use the HTTP JSON API as that returns more friendly errors than
    // the JSON-RPC API.
    return fetch(this._apiBaseUrl + "/" + method, {
      method: "POST",
      body: JSON.stringify(params),
      headers,
    }).then((r) => {
      if (r.status !== 200) {
        // For non-200 first read the response as text
        return r.text().then((t) => {
          let d;
          try {
            d = JSON.parse(t);
          } catch (e) {
            if ("DEBUG" in process.env) {
              console.log(e);
            }
            // If JSON parsing fails then throw an error with the
            // response text as the message
            throw new Error(t);
          }
          // Otherwise throw the parsed JSON error response
          // We override toString as otherwise you get expect errors like:
          //   `expected to resolve but rejected with "[object Object]"`
          Object.defineProperty(d, "toString", {
            value: () => t,
            enumerable: false,
          });
          throw d;
        });
      }

      if (this._parseJsonResult) {
        return r.text().then((t) => {
          return JSON.parse(t, reviver);
        });
      }
    });
  }
}

const dateFormat = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:(\d{2}(?:\.\d*))Z$/;

function reviver(key, value) {
  if (typeof value === "string" && dateFormat.test(value)) {
    return new Date(value);
  }
  return value;
}
