"use client";

import { FormEvent, useState } from "react";
import { useAppStore } from "@/store/appStore";

type MarketFormProps = {
  onCreated?: () => void;
};

export default function MarketForm({ onCreated }: MarketFormProps) {
  const [question, setQuestion] = useState("");
  const [description, setDescription] = useState("");
  const [resolutionDatetime, setResolutionDatetime] = useState("");
  const [option1, setOption1] = useState("");
  const [option2, setOption2] = useState("");
  const [status, setStatus] = useState<"idle" | "submitting" | "success" | "error">("idle");
  const [error, setError] = useState("");
  const { loadMarkets } = useAppStore();

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setStatus("submitting");
    setError("");

    try {
      const response = await fetch("http://localhost:8080/api/markets", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          question,
          description,
          end_time: resolutionDatetime,
          options: [option1, option2],
        }),
      });

      if (!response.ok) {
        throw new Error(`Request failed (${response.status})`);
      }

      setStatus("success");
      setQuestion("");
      setDescription("");
      setResolutionDatetime("");
      setOption1("");
      setOption2("");
      await loadMarkets();
      onCreated?.();
    } catch (err) {
      setStatus("error");
      setError(err instanceof Error ? err.message : "Unknown error");
    }
  };

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4">
      <label className="flex flex-col gap-1">
        <span>Question</span>
        <input
          type="text"
          value={question}
          onChange={(e) => setQuestion(e.target.value)}
          required
          className="rounded border px-3 py-2"
          placeholder=""
        />
      </label>

      <label className="flex flex-col gap-1">
        <span>Description</span>
        <textarea
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          required
          className="rounded border px-3 py-2"
          placeholder=""
          rows={3}
        />
      </label>

      <label className="flex flex-col gap-1">
        <span>Option 1</span>
        <input
          type="text"
          value={option1}
          onChange={(e) => setOption1(e.target.value)}
          required
          className="rounded border px-3 py-2"
        />
      </label>

      <label className="flex flex-col gap-1">
        <span>Option 2</span>
        <input
          type="text"
          value={option2}
          onChange={(e) => setOption2(e.target.value)}
          required
          className="rounded border px-3 py-2"
        />
      </label>

      <label className="flex flex-col gap-1">
        <span>Resolution datetime</span>
        <input
          type="datetime-local"
          value={resolutionDatetime ? resolutionDatetime.slice(0, 16) : ""}
          onChange={(e) => setResolutionDatetime(new Date(e.target.value).toISOString())}
          required
          className="rounded border px-3 py-2"
        />
      </label>

      <button
        type="submit"
        className="rounded bg-blue-600 px-4 py-2 text-white disabled:opacity-60"
        disabled={status === "submitting"}
      >
        {status === "submitting" ? "Creating..." : "Create market"}
      </button>

      {status === "success" && <p className="text-green-600">Market created.</p>}
      {status === "error" && <p className="text-red-600">Failed: {error}</p>}
    </form>
  );
}
