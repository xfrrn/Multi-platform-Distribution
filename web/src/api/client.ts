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
  updated_at: string;
  archived_at?: string;
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
  updated_at: string;
  archived_at?: string;
};

export type Artifact = {
  id: string;
  release_id: string;
  platform: string;
  arch: string;
  file_type: string;
  file_name: string;
  file_url: string;
  storage_key: string;
  file_size: number;
  sha512: string;
  source_type: "managed" | "anyshare";
  anyshare_docid?: string;
  anyshare_rev?: string;
  anyshare_name?: string;
  created_at: string;
  updated_at: string;
  archived_at?: string;
};

export type ServerConfig = {
  anyshare_enabled: boolean;
  public_base_url: string;
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

export type StatsPoint = {
  date: string;
  count: number;
};

export type StatsBreakdown = {
  key: string;
  count: number;
};

export type StatsSummary = {
  today_update_requests: number;
  today_downloads: number;
  downloads_7d: number;
  active_clients_7d: number;
  staging_hits_7d: number;
  forced_downloads_7d: number;
  total_update_requests: number;
  total_downloads: number;
  update_trend: StatsPoint[];
  download_trend: StatsPoint[];
  platform_breakdown: StatsBreakdown[];
  arch_breakdown: StatsBreakdown[];
  channel_breakdown: StatsBreakdown[];
  version_breakdown: StatsBreakdown[];
};

export type UpdateRequestEvent = {
  id: string;
  app_id?: string;
  app_slug: string;
  release_id?: string;
  version: string;
  channel: string;
  platform: string;
  arch: string;
  client_id: string;
  format: string;
  matched: boolean;
  staged_hit: boolean;
  ip: string;
  user_agent: string;
  created_at: string;
};

export type DownloadEvent = {
  id: string;
  app_id: string;
  release_id: string;
  artifact_id: string;
  version: string;
  platform: string;
  arch: string;
  file_type: string;
  file_name: string;
  client_id: string;
  ip: string;
  user_agent: string;
  created_at: string;
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

export type UploadProgress = {
  loaded: number;
  total: number;
  percent: number;
  lengthComputable: boolean;
};

export type ServerUploadProgress = {
  id: string;
  phase: string;
  loaded: number;
  total: number;
  percent: number;
  status: "processing" | "done" | "error";
  error?: string;
  updated_at: string;
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

  async getConfig(): Promise<ServerConfig> {
    return this.request<ServerConfig>("/api/config");
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
    icon_url?: string;
  }): Promise<DesktopApp> {
    return this.request<DesktopApp>("/api/apps", {
      method: "POST",
      body: JSON.stringify(payload)
    });
  }

  async updateApp(appId: string, payload: {
    name: string;
    description: string;
    icon_url: string;
    default_channel: string;
  }): Promise<DesktopApp> {
    return this.request<DesktopApp>(`/api/apps/${appId}`, {
      method: "PATCH",
      body: JSON.stringify(payload)
    });
  }

  async archiveApp(appId: string): Promise<void> {
    await this.requestVoid(`/api/apps/${appId}`, { method: "DELETE" });
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

  async updateRelease(releaseId: string, payload: {
    version: string;
    channel: string;
    changelog: string;
    is_forced: boolean;
    staging_percent: number;
    published_at?: string;
  }): Promise<Release> {
    return this.request<Release>(`/api/releases/${releaseId}`, {
      method: "PATCH",
      body: JSON.stringify(payload)
    });
  }

  async archiveRelease(releaseId: string): Promise<void> {
    await this.requestVoid(`/api/releases/${releaseId}`, { method: "DELETE" });
  }

  async listArtifacts(releaseId: string): Promise<Artifact[]> {
    const result = await this.request<ListResponse<Artifact>>(`/api/releases/${releaseId}/artifacts`);
    return result.items ?? [];
  }

  async uploadArtifact(releaseId: string, payload: {
    platform: string;
    arch: string;
    file_type: string;
    source_type?: "managed" | "anyshare";
    upload_id?: string;
    file: File;
  }, onProgress?: (progress: UploadProgress) => void): Promise<Artifact> {
    const body = new FormData();
    body.set("platform", payload.platform);
    body.set("arch", payload.arch);
    body.set("file_type", payload.file_type);
    if (payload.source_type) {
      body.set("source_type", payload.source_type);
    }
    if (payload.upload_id) {
      body.set("upload_id", payload.upload_id);
    }
    body.set("file", payload.file);

    return this.requestUpload<Artifact>(`/api/releases/${releaseId}/artifacts`, "POST", body, onProgress);
  }

  async getUploadProgress(uploadId: string): Promise<ServerUploadProgress> {
    return this.request<ServerUploadProgress>(`/api/uploads/${encodeURIComponent(uploadId)}`);
  }

  async updateArtifact(artifactId: string, payload: {
    platform: string;
    arch: string;
    file_type: string;
    file_name: string;
  }): Promise<Artifact> {
    return this.request<Artifact>(`/api/artifacts/${artifactId}`, {
      method: "PATCH",
      body: JSON.stringify(payload)
    });
  }

  async replaceArtifactFile(artifactId: string, file: File): Promise<Artifact> {
    const body = new FormData();
    body.set("file", file);
    return this.request<Artifact>(`/api/artifacts/${artifactId}/file`, {
      method: "PUT",
      body
    });
  }

  async archiveArtifact(artifactId: string): Promise<void> {
    await this.requestVoid(`/api/artifacts/${artifactId}`, { method: "DELETE" });
  }

  async getManifest(slug: string, params: URLSearchParams): Promise<UpdateManifest> {
    const query = params.toString();
    return this.request<UpdateManifest>(`/api/latest/${slug}/update.json${query ? `?${query}` : ""}`);
  }

  async getStatsSummary(params = new URLSearchParams()): Promise<StatsSummary> {
    return this.request<StatsSummary>(withQuery("/api/stats/summary", params));
  }

  async getAppStatsSummary(appId: string, params = new URLSearchParams()): Promise<StatsSummary> {
    return this.request<StatsSummary>(withQuery(`/api/apps/${appId}/stats/summary`, params));
  }

  async getReleaseStats(releaseId: string, params = new URLSearchParams()): Promise<StatsSummary> {
    return this.request<StatsSummary>(withQuery(`/api/releases/${releaseId}/stats`, params));
  }

  async listUpdateRequests(appId?: string, params = new URLSearchParams()): Promise<UpdateRequestEvent[]> {
    const path = appId ? `/api/apps/${appId}/stats/update-requests` : "/api/stats/update-requests";
    const result = await this.request<ListResponse<UpdateRequestEvent>>(withQuery(path, params));
    return result.items ?? [];
  }

  async listDownloads(appId?: string, params = new URLSearchParams()): Promise<DownloadEvent[]> {
    const path = appId ? `/api/apps/${appId}/stats/downloads` : "/api/stats/downloads";
    const result = await this.request<ListResponse<DownloadEvent>>(withQuery(path, params));
    return result.items ?? [];
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

  private async requestVoid(path: string, init: RequestInit = {}): Promise<void> {
    const headers = new Headers(init.headers);
    for (const [key, value] of Object.entries(this.authHeaders())) {
      headers.set(key, value);
    }
    const response = await fetch(path, { ...init, headers });
    if (!response.ok) {
      throw new ApiError(response.status, await this.errorMessage(response));
    }
  }

  private requestUpload<T>(
    path: string,
    method: string,
    body: FormData,
    onProgress?: (progress: UploadProgress) => void
  ): Promise<T> {
    return new Promise((resolve, reject) => {
      const xhr = new XMLHttpRequest();
      xhr.open(method, path);
      for (const [key, value] of Object.entries(this.authHeaders())) {
        xhr.setRequestHeader(key, value);
      }

      xhr.upload.onprogress = (event) => {
        const total = event.lengthComputable ? event.total : 0;
        const percent = total > 0 ? Math.min(100, Math.round((event.loaded / total) * 100)) : 0;
        onProgress?.({
          loaded: event.loaded,
          total,
          percent,
          lengthComputable: event.lengthComputable
        });
      };

      xhr.onload = () => {
        const parsed = parseJSON(xhr.responseText);
        if (xhr.status >= 200 && xhr.status < 300) {
          resolve(parsed as T);
          return;
        }
        reject(new ApiError(xhr.status, responseError(parsed, xhr.statusText)));
      };

      xhr.onerror = () => reject(new ApiError(0, "上传失败，请检查网络连接"));
      xhr.onabort = () => reject(new ApiError(0, "上传已取消"));
      xhr.send(body);
    });
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

function withQuery(path: string, params: URLSearchParams): string {
  const query = params.toString();
  return `${path}${query ? `?${query}` : ""}`;
}

function parseJSON(value: string): unknown {
  if (!value) return null;
  try {
    return JSON.parse(value);
  } catch {
    return value;
  }
}

function responseError(value: unknown, fallback: string): string {
  if (value && typeof value === "object" && "error" in value) {
    const message = (value as { error?: unknown }).error;
    if (typeof message === "string") return message;
  }
  if (typeof value === "string" && value) return value;
  return fallback;
}
