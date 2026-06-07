import { createContext } from "react";
import type { ArtifactUploadTask } from "../lib/types";

type UploadTasksContextValue = {
  tasks: ArtifactUploadTask[];
  add: (task: ArtifactUploadTask) => void;
  update: (id: string, patch: Partial<ArtifactUploadTask>) => void;
};

export const UploadTasksContext = createContext<UploadTasksContextValue>(null!);
