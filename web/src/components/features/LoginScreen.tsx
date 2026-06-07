import { KeyRound } from "lucide-react";
import { type FormEvent, useState } from "react";
import { ApiClient } from "../../api";
import { readError } from "../../lib/utils";
import { Button } from "../ui/Button";
import { Field } from "../ui/Field";
import "./LoginScreen.css";

export function LoginScreen({ api, onLogin }: { api: ApiClient; onLogin: (token: string) => void }) {
  const [email, setEmail] = useState("admin@example.com");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setError("");
    setSubmitting(true);
    try {
      const result = await api.login(email, password);
      onLogin(result.access_token);
    } catch (err) {
      setError(readError(err));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="login-page">
      <form className="login-panel" onSubmit={submit}>
        <div className="login-icon"><KeyRound size={28} /></div>
        <h1>登录管理台</h1>
        <p>使用管理员账号进入应用发布中心。</p>
        <Field label="邮箱">
          <input value={email} onChange={(e) => setEmail(e.target.value)} autoComplete="email" />
        </Field>
        <Field label="密码">
          <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} autoComplete="current-password" />
        </Field>
        {error && <div className="notice error">{error}</div>}
        <Button variant="primary" disabled={submitting} style={{ width: "100%", justifyContent: "center", minHeight: 42 }}>
          {submitting ? "登录中..." : "登录"}
        </Button>
      </form>
    </div>
  );
}
