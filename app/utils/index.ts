export const useAPI = async <T>(endpoint: string,
    params?: { [name: string]: any; }): Promise<Ref<T>> => {

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

// Set up the "roving tabindex" accessibility strategy. A single element in the
// container will be selectable (tabIndex=0) while the others will not
// (tabIndex=-1); the left and right keys change which element is selected.
//
// The parent container should have the `stop` class, the returned keydown
// handler should be registered on it, and tab indexes should be set according
// to the returned index variable.
export const useRovingTabIndex = (limit: number, start: number = 0):
    [FocusInfo, (e: KeyboardEvent) => void] => {

    const state = reactive({ index: start });
    const keydown = (e: KeyboardEvent) => {
        if (e.key == "ArrowRight") {
            if (state.index < limit - 1) state.index += 1;
        } else if (e.key == "ArrowLeft") {
            if (state.index > 0) state.index -= 1;
        } else {
            return;
        }
        const parent = getStopParent(document.activeElement);
        // @ts-ignore
        setTimeout(() => parent?.querySelector("[tabindex='0']")?.focus(), 0);
        e.preventDefault();
    };
    return [state, keydown];
};


// Gets the nearest parent of the given element that has the class "stop".
export const getStopParent = (element: Element | null): Element | null => {
    if (!element) {
        return null;
    } else if (element.classList.contains("stop")) {
        return element;
    } else {
        return getStopParent(element.parentElement);
    }
};

export const tabIndex = (focused: FocusInfo, target: number): number =>
    focused.index == target ? 0 : -1;
