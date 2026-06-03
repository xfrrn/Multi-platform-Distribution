export type Admin = {
  id: string;
  email: string;
  name: string;
  is_active: boolean;
  created_at: string;
};

export type DesktopApp = {
  id: string;
  name: string;
  slug: string;
  description: string;
  icon_url: string;
  default_channel: string;
  created_at: string;
};

export type Release = {
  id: string;
  app_id: string;
  version: string;
  channel: string;
  changelog: string;
  is_forced: boolean;
  staging_percent: number;
  published_at: string;
  created_at: string;
};

export type Artifact = {
  id: string;
  release_id: string;
  platform: string;
  arch: string;
  file_type: string;
  file_url: string;
  storage_key: string;
  file_size: number;
  sha512: string;
  created_at: string;
};

export type UpdateManifest = {
  version: string;
  channel: string;
  changelog: string;
  forced: boolean;
  staging_percent: number;
  published_at: string;
  files: Array<{
    platform: string;
    arch: string;
    type: string;
    url: string;
    size: number;
    sha512: string;
  }>;
};

type LoginResponse = {
  access_token: string;
  token_type: string;
  expires_at: string;
  admin: Admin;
};

type ListResponse<T> = {
  items: T[];
};

export class ApiError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

export class ApiClient {
  private token: string | null;

  constructor(token: string | null) {
    this.token = token;
  }

  setToken(token: string | null) {
    this.token = token;
  }

  async login(email: string, password: string): Promise<LoginResponse> {
    return this.request<LoginResponse>("/api/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password })
    });
  }

  async listApps(): Promise<DesktopApp[]> {
    const result = await this.request<ListResponse<DesktopApp>>("/api/apps");
    return result.items ?? [];
  }

  async createApp(payload: {
    name: string;
    slug: string;
    description: string;
    default_channel: string;
  }): Promise<DesktopApp> {
    return this.request<DesktopApp>("/api/apps", {
      method: "POST",
      body: JSON.stringify(payload)
    });
  }

  async listReleases(appId: string): Promise<Release[]> {
    const result = await this.request<ListResponse<Release>>(`/api/apps/${appId}/releases`);
    return result.items ?? [];
  }

  async createRelease(appId: string, payload: {
    version: string;
    channel: string;
    changelog: string;
    is_forced: boolean;
    staging_percent: number;
    published_at?: string;
  }): Promise<Release> {
    return this.request<Release>(`/api/apps/${appId}/releases`, {
      method: "POST",
      body: JSON.stringify(payload)
    });
  }

  async uploadArtifact(releaseId: string, payload: {
    platform: string;
    arch: string;
    file_type: string;
    file: File;
  }): Promise<Artifact> {
    const body = new FormData();
    body.set("platform", payload.platform);
    body.set("arch", payload.arch);
    body.set("file_type", payload.file_type);
    body.set("file", payload.file);

    return this.request<Artifact>(`/api/releases/${releaseId}/artifacts`, {
      method: "POST",
      body
    });
  }

  async getManifest(slug: string, params: URLSearchParams): Promise<UpdateManifest> {
    const query = params.toString();
    return this.request<UpdateManifest>(`/api/latest/${slug}/update.json${query ? `?${query}` : ""}`);
  }

  async getText(path: string): Promise<string> {
    const response = await fetch(path, { headers: this.authHeaders() });
    if (!response.ok) {
      throw new ApiError(response.status, await this.errorMessage(response));
    }
    return response.text();
  }

  private async request<T>(path: string, init: RequestInit = {}): Promise<T> {
    const headers = new Headers(init.headers);
    const isFormData = init.body instanceof FormData;
    if (!isFormData && init.body) {
      headers.set("Content-Type", "application/json");
    }
    for (const [key, value] of Object.entries(this.authHeaders())) {
      headers.set(key, value);
    }

    const response = await fetch(path, { ...init, headers });
    if (!response.ok) {
      throw new ApiError(response.status, await this.errorMessage(response));
    }
    return response.json() as Promise<T>;
  }

  private authHeaders(): Record<string, string> {
    return this.token ? { Authorization: `Bearer ${this.token}` } : {};
  }

  private async errorMessage(response: Response): Promise<string> {
    try {
      const body = await response.json();
      return body.error || response.statusText;
    } catch {
      return response.statusText;
    }
  }
}
