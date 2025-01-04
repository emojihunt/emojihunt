export function highlightContents(element: HTMLElement): void {
  const node = element.childNodes[0];
  const range = document.createRange();
  range.setStart(node, 0);
  range.setEnd(node, node.textContent?.length || 0);
  const selection = window.getSelection();
  selection?.removeAllRanges();
  selection?.addRange(range);
}

export const timelineFromSequence = (id: number): string => `--round-${id}`;
