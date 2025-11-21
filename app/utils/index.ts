export const highlightContents = (element: HTMLElement): void => {
  const node = element.childNodes[0]!;
  const range = document.createRange();
  range.setStart(node, 0);
  range.setEnd(node, node.textContent?.length || 0);
  const selection = window.getSelection();
  selection?.removeAllRanges();
  selection?.addRange(range);
};

export const timelineFromSequence = (id: number): string => `--round-${id}`;

export const formSubmit = async (
  url: string,
  data: object,
  method: "POST" | "DELETE" = "POST",
): Promise<Response> => {
  const { apiBase } = useAppConfig();
  return await fetch(apiBase + url, {
    method: method,
    credentials: "include",
    headers: {
      "Content-Type": "application/x-www-form-urlencoded",
    },
    body: (new URLSearchParams(data as any)).toString(),
  });
};
