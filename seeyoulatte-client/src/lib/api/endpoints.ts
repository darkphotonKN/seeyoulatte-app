export const API_ENDPOINTS = {
  // Auth endpoints
  AUTH: {
    LOGIN: "/api/auth/signin",
    REGISTER: "/api/auth/signup",
    LOGOUT: "/api/auth/logout",
    REFRESH: "/api/auth/refresh",
    ME: "/api/auth/me",
  },

  // Listing endpoints
  ITEM: {
    LIST: "/api/listings",
    CREATE: "/api/listings",
    GET: (id: string | number) => `/api/listings/${id}`,
    UPDATE: (id: string | number) => `/api/listings/${id}`,
    DELETE: (id: string | number) => `/api/listings/${id}`,
  },

  // Upload endpoints
  UPLOAD: {
    IMAGE: "/upload/image",
    FILE: "/upload/file",
    PRESIGNED_URL: "/upload/presigned-url",
  },
} as const;