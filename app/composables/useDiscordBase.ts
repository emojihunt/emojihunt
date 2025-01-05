// Remembers whether the user logged in via the app or website, and provides a
// URL base accordingly.
export default function (): [string, string] {
  const discord = useCookie("discord");
  if (discord.value === "app") {
    return ["discord://", ""];
  } else {
    return ["https://discord.com", "_blank"];
  }
};
