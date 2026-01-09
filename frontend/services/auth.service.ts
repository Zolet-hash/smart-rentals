import { api } from "./api";

export type LoginPayload = {
  email: string;
  password: string;
};

export type RegisterPayload = {
  name: string;
  email: string;
  password: string;
};

export const authService = {
  login: (data: LoginPayload) => api.post("/api/users/login", data),
  register: (data: RegisterPayload) => api.post("/api/users/register", data),
  me: () => api.get("/api/users/id"),
};
