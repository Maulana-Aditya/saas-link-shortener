import { createContext, useContext, useEffect, useState, type ReactNode } from "react";
import { api, clearTokens, getAccessToken, setTokens } from "./api";

type User = { id: string; email: string; name: string };
type Org = { id: string; name: string; plan: string };

type AuthContextValue = {
  user: User | null;
  org: Org | null;
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  signup: (email: string, password: string, name: string, orgName?: string) => Promise<void>;
  logout: () => void;
  refreshMe: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

type AuthResponse = {
  access_token: string;
  refresh_token: string;
  user: User;
  org: Org;
};

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [org, setOrg] = useState<Org | null>(null);
  const [loading, setLoading] = useState(true);

  const refreshMe = async () => {
    const data = await api.get<{ user: User; org: Org }>("/api/v1/me");
    setUser(data.user);
    setOrg(data.org);
  };

  useEffect(() => {
    if (!getAccessToken()) {
      setLoading(false);
      return;
    }
    refreshMe()
      .catch(() => {
        clearTokens();
        setUser(null);
        setOrg(null);
      })
      .finally(() => setLoading(false));
  }, []);

  const login = async (email: string, password: string) => {
    const data = await api.post<AuthResponse>("/api/v1/auth/login", { email, password });
    setTokens(data.access_token, data.refresh_token);
    setUser(data.user);
    setOrg(data.org);
  };

  const signup = async (email: string, password: string, name: string, orgName?: string) => {
    const data = await api.post<AuthResponse>("/api/v1/auth/signup", {
      email,
      password,
      name,
      org_name: orgName,
    });
    setTokens(data.access_token, data.refresh_token);
    setUser(data.user);
    setOrg(data.org);
  };

  const logout = () => {
    clearTokens();
    setUser(null);
    setOrg(null);
  };

  return (
    <AuthContext.Provider value={{ user, org, loading, login, signup, logout, refreshMe }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
