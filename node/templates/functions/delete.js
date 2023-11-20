function deleteFunction({ model, whereInputs }) {
  return function (hooks = {}) {
    return async function (ctx, inputs) {
      let wheres = {};
      for (const key of whereInputs) {
        if (key in inputs) {
          wheres[key] = inputs[key];
        }
      }

      let data = model.where(wheres);

      if (hooks.beforeQuery) {
        data = await runtime.tracing.withSpan("beforeQuery", () => {
          return hooks.beforeQuery(ctx, inputs, data);
        });
      }

      const constructor = data?.constructor?.name;
      if (constructor === "QueryBuilder") {
        data = await data.findOne();
      }

      if (data === null) {
        throw new NoResultError();
      }

      if (hooks.beforeWrite) {
        await runtime.tracing.withSpan("beforeWrite", () => {
          return hooks.beforeWrite(ctx, inputs, data);
        });
      }

      await model.delete({ id: data.id });

      if (hooks.afterWrite) {
        await runtime.tracing.withSpan("afterWrite", () => {
          return hooks.afterWrite(ctx, inputs, data);
        });
      }

      return data.id;
    };
  };
}
