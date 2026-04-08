import { useEffect, useRef, useState } from "react";

const API_BASE = "/api";
const POLL_INTERVAL_MS = 2500;

const initialForm = {
  mode: "upload",
};

function formatDate(value) {
  if (!value) return "pending";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}

function formatDuration(value) {
  if (typeof value !== "number" || Number.isNaN(value)) return "n/a";
  if (value < 1000) return `${value} ms`;
  return `${(value / 1000).toFixed(2)} s`;
}

function statusTone(status) {
  if (status === "succeeded") return "bg-emerald-100 text-emerald-900";
  if (status === "failed") return "bg-red-100 text-red-900";
  if (status === "running") return "bg-amber-100 text-amber-900";
  return "bg-slate-200 text-slate-800";
}

async function parseJSON(response) {
  const payload = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new Error(payload.error || `request failed with status ${response.status}`);
  }
  return payload;
}

export default function App() {
  const [form, setForm] = useState(initialForm);
  const [selectedFile, setSelectedFile] = useState(null);
  const [health, setHealth] = useState(null);
  const [job, setJob] = useState(null);
  const [currentJobId, setCurrentJobId] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState("");
  const pollTimerRef = useRef(null);

  useEffect(() => {
    fetch(`${API_BASE}/health`)
      .then(parseJSON)
      .then(setHealth)
      .catch((err) => {
        setError(err.message);
      });
  }, []);

  useEffect(() => {
    if (!currentJobId) return undefined;

    let cancelled = false;

    const pollJob = async () => {
      try {
        const payload = await parseJSON(await fetch(`${API_BASE}/generate/${currentJobId}`));
        if (cancelled) return;
        setJob(payload);
        if (payload.status === "running" || payload.status === "pending") {
          pollTimerRef.current = window.setTimeout(pollJob, POLL_INTERVAL_MS);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err.message);
        }
      }
    };

    pollJob();

    return () => {
      cancelled = true;
      if (pollTimerRef.current) {
        window.clearTimeout(pollTimerRef.current);
      }
    };
  }, [currentJobId]);

  const resetFlow = () => {
    if (pollTimerRef.current) {
      window.clearTimeout(pollTimerRef.current);
    }
    setForm(initialForm);
    setSelectedFile(null);
    setJob(null);
    setCurrentJobId("");
    setError("");
    setIsSubmitting(false);
  };

  const onSubmit = async (event) => {
    event.preventDefault();
    setError("");
    setIsSubmitting(true);
    setJob(null);

    try {
      let response;
      if (!selectedFile) {
        throw new Error("select an OpenAPI file before generating");
      }

      const formData = new FormData();
      formData.append("specFile", selectedFile);
      response = await fetch(`${API_BASE}/generate`, {
        method: "POST",
        body: formData,
      });

      const jobPayload = await parseJSON(response);
      setCurrentJobId(jobPayload.jobId);
      setJob(jobPayload);
    } catch (err) {
      setError(err.message);
    } finally {
      setIsSubmitting(false);
    }
  };

  const downloadUrl = job?.jobId && job.status === "succeeded" ? `${API_BASE}/generate/${job.jobId}/download` : "";
  const showSuccessDetails = job?.status === "succeeded";

  return (
    <main className="min-h-screen bg-haze font-body text-ink">
      <div className="mx-auto flex min-h-screen max-w-[96rem] flex-col gap-8 px-4 py-6 sm:px-6 lg:px-10">
        <header className="rounded-[2rem] border border-white/70 bg-white/70 p-6 shadow-panel backdrop-blur">
          <div className="w-full space-y-4">
            <div className="inline-flex rounded-full border border-ink/10 bg-white/80 px-3 py-1 text-xs font-semibold uppercase tracking-[0.22em] text-leaf">
              OpenAPI to Go SDK
            </div>
            <div className="space-y-3">
              <h1 className="w-full font-display text-4xl font-semibold leading-tight sm:text-5xl xl:text-[4.5rem] xl:leading-[1.02]">
                Turn OpenAPI specs into Go integration packages your team can review, test, and ship faster.
              </h1>
              <p className="w-full text-sm leading-6 text-ink/70 sm:text-base xl:text-lg xl:leading-8">
                Upload a spec, run the generation flow, track progress live, and pull the generated ZIP when the package is ready.
              </p>
            </div>
          </div>
        </header>

        <section className="grid gap-8 xl:grid-cols-[0.95fr_1.05fr]">
          {!job ? (
            <form
              onSubmit={onSubmit}
              className="space-y-6 rounded-[2rem] border border-white/80 bg-white/80 p-6 shadow-panel backdrop-blur"
            >
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <h2 className="font-display text-2xl font-semibold">Start a generation</h2>
                  <p className="mt-1 text-sm text-ink/65">Upload an OpenAPI file and send it to the backend generation flow.</p>
                </div>
              </div>

              <label className="block rounded-[1.5rem] border border-dashed border-ink/20 bg-mist p-5 transition hover:border-coral/60">
                <span className="block text-sm font-semibold text-ink">OpenAPI file</span>
                <span className="mt-1 block text-sm text-ink/60">JSON or YAML. The backend stores the uploaded file in /tmp.</span>
                <input
                  type="file"
                  accept=".json,.yaml,.yml"
                  className="mt-4 block w-full text-sm text-ink file:mr-4 file:rounded-full file:border-0 file:bg-coral file:px-4 file:py-2 file:font-semibold file:text-white hover:file:bg-[#dd6345]"
                  onChange={(event) => setSelectedFile(event.target.files?.[0] || null)}
                />
                {selectedFile ? (
                  <p className="mt-3 font-mono text-xs text-leaf">
                    {selectedFile.name} · {(selectedFile.size / 1024).toFixed(1)} KB
                  </p>
                ) : null}
              </label>

              {error ? <div className="rounded-2xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">{error}</div> : null}

              <button
                type="submit"
                disabled={isSubmitting}
                className="inline-flex w-full items-center justify-center rounded-full bg-ink px-5 py-3 text-sm font-semibold text-white transition hover:bg-leaf disabled:cursor-not-allowed disabled:bg-ink/40"
              >
                {isSubmitting ? "Submitting..." : "Generate SDK"}
              </button>
            </form>
          ) : (
            <section className="space-y-6 rounded-[2rem] border border-white/80 bg-white/80 p-6 shadow-panel backdrop-blur">
              <div className="rounded-[1.5rem] border border-ink/10 bg-mist/80 p-5">
                <h2 className="font-display text-2xl font-semibold">Your generation is in motion</h2>
                <p className="mt-2 text-sm leading-6 text-ink/65">
                  Follow the job on the right. When you want to run a new spec, reset this session and upload another file.
                </p>
                <button
                  type="button"
                  onClick={resetFlow}
                  className="mt-4 inline-flex rounded-full border border-ink/15 bg-white px-4 py-2 text-sm font-semibold text-ink transition hover:border-ink/35 hover:bg-ink hover:text-white"
                >
                  Start a new generation
                </button>
              </div>
            </section>
          )}

          {job ? (
            <section className="space-y-6 rounded-[2rem] border border-white/80 bg-white/80 p-6 shadow-panel backdrop-blur">
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div>
                <h2 className="font-display text-2xl font-semibold">Job console</h2>
                <p className="mt-1 text-sm text-ink/65">Track status, inspect previews, and download the generated archive.</p>
              </div>
              {job?.status ? (
                <span className={`rounded-full px-3 py-1 text-xs font-semibold uppercase tracking-[0.16em] ${statusTone(job.status)}`}>
                  {job.status}
                </span>
              ) : null}
            </div>

            {!job ? (
              <div className="rounded-[1.5rem] border border-dashed border-ink/15 bg-mist/60 p-10 text-center text-sm text-ink/55">
                No job yet. Submit a spec on the left to start polling the backend.
              </div>
            ) : (
              <div className="space-y-6">
                {showSuccessDetails ? (
                  <>
                    <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
                      <MetricCard label="Created" value={formatDate(job.createdAt)} />
                      <MetricCard label="Started" value={formatDate(job.startedAt)} />
                      <MetricCard label="Completed" value={formatDate(job.completedAt)} />
                      <MetricCard label="Files" value={String(job.files?.length ?? 0)} />
                    </div>

                    {job.metrics ? (
                      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-2">
                        <MetricCard label="Request time" value={formatDuration(job.metrics.requestDurationMs)} />
                        <MetricCard label="Model" value={job.metrics.model || "n/a"} mono />
                      </div>
                    ) : null}
                  </>
                ) : null}

                {job.error ? (
                  <div className="rounded-2xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">{job.error}</div>
                ) : null}

                {showSuccessDetails ? (
                  <div className="rounded-[1.5rem] bg-ink p-5 text-mist">
                    <h3 className="font-display text-lg font-semibold">Artifacts</h3>
                    <p className="mt-1 text-sm text-mist/60">Generated files reported by the backend.</p>
                    <div className="mt-4 space-y-2">
                      {job.files?.length ? (
                        job.files.map((file) => (
                          <div key={file.path} className="rounded-2xl bg-white/5 px-4 py-3">
                            <div className="font-mono text-xs text-cyan">{file.path}</div>
                            <div className="mt-1 text-xs text-mist/55">{file.bytes} bytes</div>
                          </div>
                        ))
                      ) : null}
                    </div>
                    {downloadUrl ? (
                      <a
                        href={downloadUrl}
                        className="mt-5 inline-flex rounded-full bg-coral px-5 py-3 text-sm font-semibold text-white transition hover:bg-[#dd6345]"
                      >
                        Download generated ZIP
                      </a>
                    ) : null}
                  </div>
                ) : null}
              </div>
            )}
            </section>
          ) : null}
        </section>
      </div>
    </main>
  );
}

function MetricCard({ label, value, mono = false }) {
  return (
    <div className="rounded-[1.5rem] border border-ink/10 bg-white p-4">
      <div className="text-xs uppercase tracking-[0.16em] text-ink/45">{label}</div>
      <div className={`mt-2 text-sm font-semibold text-ink ${mono ? "font-mono text-xs" : ""}`}>{value}</div>
    </div>
  );
}
