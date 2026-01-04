import type { UseFetchOptions } from "#app";
import type { FetchError } from "ofetch";
import type { Ref } from "vue";

export default function <T>(
  url: string,
  opts?: UseFetchOptions<T>,
): Promise<{
  data: Ref<T | null>;
  error: Ref<FetchError | null>;
}> {
  const { apiBase } = useAppConfig();
  let params = {
    baseURL: apiBase,
    credentials: "include",
    timeout: 10_000,
    ...(opts as any),
  };

  const cookie = useRequestHeader("cookie");
  if (import.meta.server && cookie) {
    params.headers = {
      cookie, ...params.headers
    };
  }
  return useFetch<T>(url, params) as any;
};
