// Set up an IntersectionObserver to detect when elements with `position:
// sticky` sticks to the top of the page. The observer signals sticking by
// adding or removing the `stuck` class from the given element.
//
// Note: this helper does not work on Chrome if the element has a drop-shadow
// filter. See: crbug.com/1358819.
//
export default function (margin: number): IntersectionObserver {
  const callback: IntersectionObserverCallback = (entries) => {
    // We get events when the element touches or un-touches the header *and*
    // when it enters or exits the viewport from below. Check the
    // y-coordinate to disambiguate.
    for (const { boundingClientRect, isIntersecting, target } of entries) {
      if (!isIntersecting && boundingClientRect.y < (margin + 1)) {
        let found = false;
        for (const title of document.querySelectorAll(".titles")) {
          if (!found) title.classList.add("stuck");
          else title.classList.remove("stuck");
          if (title === target) found = true;
        }
      } else if (isIntersecting && boundingClientRect.y < (margin + 15)) {
        let found = false;
        for (const title of document.querySelectorAll(".titles")) {
          if (title === target) found = true;
          if (!found) title.classList.add("stuck");
          else title.classList.remove("stuck");
        }
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
