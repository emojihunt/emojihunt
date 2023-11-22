export const useAPI = async (endpoint: string, params?: { [name: string]: any; }): Promise<any> => {
    if (!endpoint.startsWith("/")) {
        throw `endpoint missing leading slash: "${endpoint}"`;
    }
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
