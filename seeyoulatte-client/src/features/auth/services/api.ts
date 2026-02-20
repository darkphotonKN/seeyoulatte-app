import { apiClient } from "@/lib/api/client";

export interface SignUpRequest {
  email: string;
  password: string;
  name: string;
}

export interface SignInRequest {
  email: string;
  password: string;
}

export interface AuthResponse {
  user: {
    id: string;
    email: string;
    name: string;
    bio?: string;
    location_text?: string;
    avatar_url?: string;
    is_verified: boolean;
    created_at: string;
  };
  token: string;
}

export const authService = {
  signUp: async (data: SignUpRequest): Promise<AuthResponse> => {
    const response = await apiClient.post("/api/auth/signup", data);
    return response.data;
  },

  signIn: async (data: SignInRequest): Promise<AuthResponse> => {
    const response = await apiClient.post("/api/auth/signin", data);
    return response.data;
  },

  googleAuth: async (idToken: string): Promise<AuthResponse> => {
    const response = await apiClient.post("/api/auth/google", {
      id_token: idToken,
    });
    return response.data;
  },

  getCurrentUser: async () => {
    const response = await apiClient.get("/api/auth/me");
    return response.data;
  },
};