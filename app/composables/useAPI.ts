export default async function <T>(
  endpoint: string, params?: { [name: string]: any; }): Promise<Ref<T>> {

  if (import.meta.server && !useCookie("session").value) {
    throw createError({
      message: "short-circuiting to login page",
      statusCode: 401,
    });
  }

  let opts = {};
  if (params) {
    opts = {
      method: "POST",
      headers: {
        "Content-Type": "application/x-www-form-urlencoded",
      },
      body: (new URLSearchParams(params)).toString(),
    };
  }
  const { data, error } = await useFetch(`/api${endpoint}`, opts);
  if (error.value) {
    throw createError({
      fatal: true,
      message: error.value.message,
      statusCode: error.value.statusCode,
    });
  }
  return data as Ref<T>;
};