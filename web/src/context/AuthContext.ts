import { createContext } from "react";
import type { ApiClient } from "../api";

export type AuthState = {
  token: string | null;
  api: ApiClient;
};

export const AuthContext = createContext<{
  token: string | null;
  api: ApiClient;
  login: (token: string) => void;
  logout: () => void;
}>(null!);
