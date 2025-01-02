// Compute the redirect URI for OAuth2 authentication.
export default function (): string {
  const url = useRequestURL();
  return (new URL("/login", url)).toString();
};
