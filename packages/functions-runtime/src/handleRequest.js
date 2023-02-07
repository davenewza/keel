const {
  createJSONRPCErrorResponse,
  createJSONRPCSuccessResponse,
  JSONRPCErrorCode,
} = require("json-rpc-2.0");

const { errorToJSONRPCResponse, RuntimeErrors } = require("./errors");

// Generic handler function that is agnostic to runtime environment (local or lambda)
// to execute a custom function based on the contents of a jsonrpc-2.0 payload object.
// To read more about jsonrpc request and response shapes, please read https://www.jsonrpc.org/specification
async function handleRequest(request, config) {
  const { createFunctionAPI, createContextAPI, functions } = config;

  if (!(request.method in functions)) {
    return createJSONRPCErrorResponse(
      request.id,
      JSONRPCErrorCode.MethodNotFound,
      `no corresponding function found for '${request.method}'`
    );
  }

  try {
    const result = await functions[request.method](
      request.params,
      createFunctionAPI(),
      createContextAPI(request.meta)
    );

    if (result === undefined) {
      // no result returned from custom function
      return createJSONRPCErrorResponse(
        request.id,
        RuntimeErrors.NoResultError,
        `no result returned from function '${request.method}'`
      );
    }

    return createJSONRPCSuccessResponse(request.id, result);
  } catch (e) {
    if (e instanceof Error) {
      return errorToJSONRPCResponse(request, e);
    }

    return createJSONRPCErrorResponse(
      request.id,
      RuntimeErrors.UnknownError,
      JSON.stringify(e)
    );
  }
}

module.exports = {
  handleRequest,
  RuntimeErrors,
};
