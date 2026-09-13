import { fetchJSON, fetchURL } from "./utils";

export interface Favorite {
  path: string;
  realPath: string;
}

export async function getFavorites(): Promise<Favorite[]> {
  return await fetchJSON<Favorite[]>("/api/favorites", {});
}

async function mutateFavorite(method: "POST" | "DELETE", path: string) {
  const res = await fetchURL("/api/favorites", {
    method,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ path }),
  });
  return (await res.json()) as Favorite[];
}

export function addFavorite(path: string) {
  return mutateFavorite("POST", path);
}

export function removeFavorite(path: string) {
  return mutateFavorite("DELETE", path);
}
