export const useAPI = async (endpoint: string): Promise<any> => {
    if (!endpoint.startsWith("/")) {
        throw `endpoint missing leading slash: "${endpoint}"`;
    }
    const { data, error } = await useFetch(`/api${endpoint}`);
    if (error.value) {
        if (error.value.statusCode == 401) {
            throw createError({ statusCode: 401 });
        }
        throw createError(error.value?.message || "API request failed");
    }
    return data;
};
