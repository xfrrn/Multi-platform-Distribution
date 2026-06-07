import { type FormEvent, useState } from "react";
import type { DesktopApp, ApiClient } from "../../api";
import { CHANNELS } from "../../lib/constants";
import { readError } from "../../lib/utils";
import { Modal, ModalActions, Field, Button } from "../ui";
import { Plus } from "lucide-react";

export function CreateAppDialog({ api, onClose, onCreated }: {
  api: ApiClient;
  onClose: () => void;
  onCreated: (app: DesktopApp) => void;
}) {
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [description, setDescription] = useState("");
  const [iconURL, setIconURL] = useState("");
  const [channel, setChannel] = useState("stable");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setError("");
    setSubmitting(true);
    try {
      const app = await api.createApp({ name, slug, description, icon_url: iconURL, default_channel: channel });
      onCreated(app);
    } catch (err) {
      setError(readError(err));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Modal title="创建应用" onClose={onClose}>
      <form onSubmit={submit} style={{ display: "grid", gap: "14px" }}>
        <Field label="应用名称">
          <input value={name} onChange={(e) => setName(e.target.value)} placeholder="例如 Desktop Client" />
        </Field>
        <Field label="Slug">
          <input value={slug} onChange={(e) => setSlug(e.target.value)} placeholder="desktop-client" />
        </Field>
        <Field label="描述">
          <textarea value={description} onChange={(e) => setDescription(e.target.value)} />
        </Field>
        <Field label="图标 URL">
          <input value={iconURL} onChange={(e) => setIconURL(e.target.value)} />
        </Field>
        <Field label="默认渠道">
          <select value={channel} onChange={(e) => setChannel(e.target.value)}>
            {CHANNELS.map((c) => <option key={c}>{c}</option>)}
          </select>
        </Field>
        {error && <div className="notice error">{error}</div>}
        <ModalActions>
          <Button variant="ghost" type="button" onClick={onClose}>取消</Button>
          <Button variant="primary" disabled={submitting}><Plus size={18} />创建</Button>
        </ModalActions>
      </form>
    </Modal>
  );
}
