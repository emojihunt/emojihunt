import { appendResponseHeader, H3Event } from "h3";

export const useAPI = async (endpoint: string, params?: { [name: string]: any; }): Promise<any> => {
    var opts = {};
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
        throw error.value;
    }
    return data;
};

type AuthenticateResponse =
    { error: null, username: string; } |
    { error: "unknown_member", username: string; } |
    { error: "invalid_code"; };


export const useAuthenticateAPI = async (event: H3Event, code: string): Promise<AuthenticateResponse> => {
    const response = await $fetch.raw("/api/authenticate", {
        method: "POST",
        headers: {
            "Content-Type": "application/x-www-form-urlencoded",
        },
        body: (new URLSearchParams({ code })).toString(),
    });
    const { username } = response._data as any;
    if (response.status == 200) {
        if (import.meta.server) {
            const cookies = (response.headers.get('set-cookie') || '');
            appendResponseHeader(event, 'set-cookie', cookies);
        }
        return { error: null, username };
    } else if (response.status == 403) {
        if (username) {
            return { error: "unknown_member", username };
        } else {
            return { error: "invalid_code" };
        }
    } else {
        throw createError({
            statusCode: response.status,
            statusMessage: response.statusText,
        });
    }
};
