// Remembers whether the user logged in via the app or website, and provides a
// URL base accordingly.
export default function (): [string, string] {
  const discord = useCookie("discord");
  if (discord.value === "app") {
    // The iOS app is flexible about the "host" in this custom URL scheme, but
    // the Android app requires "app".
    return ["discord://app", ""];
  } else {
    return ["https://discord.com", "_blank"];
  }
};
