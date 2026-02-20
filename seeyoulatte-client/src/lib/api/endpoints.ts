export const API_ENDPOINTS = {
  // Auth endpoints
  AUTH: {
    LOGIN: "/auth/login",
    REGISTER: "/auth/register",
    LOGOUT: "/auth/logout",
    REFRESH: "/auth/refresh",
    ME: "/auth/me",
  },

  // Listing endpoints
  ITEM: {
    LIST: "/listings",
    CREATE: "/listings",
    GET: (id: string | number) => `/listings/${id}`,
    UPDATE: (id: string | number) => `/listings/${id}`,
    DELETE: (id: string | number) => `/listings/${id}`,
  },

  // Upload endpoints
  UPLOAD: {
    IMAGE: "/upload/image",
    FILE: "/upload/file",
    PRESIGNED_URL: "/upload/presigned-url",
  },
} as const;