/* ── Type exports ── */
export type {
  Admin,
  DesktopApp,
  Release,
  Artifact,
  ServerConfig,
  UpdateManifest,
  StatsPoint,
  StatsBreakdown,
  StatsSummary,
  UpdateRequestEvent,
  DownloadEvent,
  UploadProgress,
  ServerUploadProgress,
} from "../api";

export type View = "home" | "detail" | "stats";
export type DetailTab = "overview" | "releases" | "artifacts" | "metadata" | "stats" | "settings";
export type MetaFormat = "json" | "yml" | "xml";
export type ArtifactSource = "managed" | "anyshare";
export type UploadTaskStatus = "uploading" | "processing" | "done" | "error";

export type Toast = { id: number; message: string; tone: "success" | "error" };

export type ArtifactUploadPayload = {
  platform: string;
  arch: string;
  file_type: string;
  source_type: ArtifactSource;
  upload_id?: string;
  file: File;
};

export type ArtifactUploadTask = {
  id: string;
  releaseId: string;
  fileName: string;
  fileSize: number;
  target: string;
  loaded: number;
  total: number;
  progress: number;
  status: UploadTaskStatus;
  serverPhase?: string;
  error?: string;
};

export type AppStats = {
  latestVersion: string;
  artifactCount: number;
  platforms: string[];
};
