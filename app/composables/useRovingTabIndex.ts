// Set up the "roving tabindex" accessibility strategy. A single element in the
// container will be selectable (tabIndex=0) while the others will not
// (tabIndex=-1); the left and right keys change which element is selected.
//
// The parent container should have the `stop` class, the returned keydown
// handler should be registered on it, and tab indexes should be set according
// to the returned index variable.
export default function (limit: number, start: number = 0):
  [FocusInfo, (e: KeyboardEvent) => void] {

  const state = reactive({ index: start });
  const keydown = (e: KeyboardEvent) => {
    if (e.key === "ArrowRight") {
      if (state.index < limit - 1) state.index += 1;
    } else if (e.key === "ArrowLeft") {
      if (state.index > 0) state.index -= 1;
    } else {
      return;
    }
    const parent = getStopParent(document.activeElement);
    // @ts-ignore
    nextTick(() => parent?.querySelector("[tabindex='0']")?.focus());
    e.preventDefault();
    e.stopPropagation();
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
  focused.index === target ? 0 : -1;
