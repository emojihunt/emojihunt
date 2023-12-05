export const loginNonceValue = "login.nonce";

export const useAPI = async <T>(endpoint: string, params?: { [name: string]: any; }): Promise<Ref<T>> => {
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

// Set up an IntersectionObserver to detect when elements with `position:
// sticky` sticks to the top of the page. The observer signals sticking by
// adding or removing the `stuck` class from the given element.
//
// Note: this helper does not work on Chrome if the element has a drop-shadow
// filter. See: crbug.com/1358819.
//
export const useStickyIntersectionObserver = (margin: number): IntersectionObserver => {
    const callback: IntersectionObserverCallback = (entries) => {
        // We get events when the element touches or un-touches the header *and*
        // when it enters or exits the viewport from below. Check the
        // y-coordinate to disambiguate.
        for (const { boundingClientRect, isIntersecting, target } of entries) {
            if (!isIntersecting && boundingClientRect.y < (margin + 1)) {
                target.classList.add("stuck");
            } else {
                target.classList.remove("stuck");
            }
        }
    };
    return new IntersectionObserver(callback, {
        root: null,
        rootMargin: `-${margin}px 0px 0px 0px`,
        threshold: 1.0,
    });
};
