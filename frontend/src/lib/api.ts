// REST API client for VectorGo

const API_BASE_URL = "/api";

export interface ProcessingResult {
    status: string;
    filename: string;
    chunkSize: number;
    chunkStride: number;
}

export interface SearchResult {
    ids: string[][];
    documents: string[][];
    metadatas: any[][];
    distances: number[][];
}

class ApiError extends Error {
    constructor(public status: number, message: string) {
        super(message);
        this.name = "ApiError";
    }
}

async function handleResponse<T>(response: Response): Promise<T> {
    if (!response.ok) {
        const text = await response.text();
        throw new ApiError(response.status, text || response.statusText);
    }
    return response.json();
}


const TOKEN_KEY = "vectorgo_token";

function getAuthHeader(): HeadersInit {
    const token = localStorage.getItem(TOKEN_KEY);
    return token ? { "Authorization": `Bearer ${token}` } : {};
}

export const api = {
    isLoggedIn(): boolean {
        return !!localStorage.getItem(TOKEN_KEY);
    },

    async login(username: string, password: string): Promise<void> {
        const response = await fetch(`${API_BASE_URL}/login`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ username, password }),
        });

        const data = await handleResponse<{ token: string }>(response);
        localStorage.setItem(TOKEN_KEY, data.token);
    },

    logout() {
        localStorage.removeItem(TOKEN_KEY);
    },

    async uploadPDF(formData: FormData): Promise<ProcessingResult> {
        const response = await fetch(`${API_BASE_URL}/upload`, {
            method: "POST",
            headers: { ...getAuthHeader() },
            body: formData,
        });
        return handleResponse<ProcessingResult>(response);
    },

    async searchVectors(query: string): Promise<SearchResult> {
        const response = await fetch(`${API_BASE_URL}/search?q=${encodeURIComponent(query)}`, {
            headers: getAuthHeader()
        });
        return handleResponse<SearchResult>(response);
    },

    async resetCollection(): Promise<{ status: string }> {
        const response = await fetch(`${API_BASE_URL}/reset`, {
            method: "POST",
            headers: getAuthHeader(),
        });
        return handleResponse<{ status: string }>(response);
    }
};
