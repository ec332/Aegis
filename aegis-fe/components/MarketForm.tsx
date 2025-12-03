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
  const { loadMarkets, loadOptionsForMarket } = useAppStore();

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
          end_time: resolutionDatetime ? new Date(resolutionDatetime).toISOString() : "",
          options: [option1, option2],
        }),
      });

      if (!response.ok) {
        throw new Error(`Request failed (${response.status})`);
      }

      const payload = await response.json();
      const createdMarketId = payload?.market?.id || payload?.id;

      setStatus("success");
      setQuestion("");
      setDescription("");
      setResolutionDatetime("");
      setOption1("");
      setOption2("");
      await loadMarkets();
      if (createdMarketId) {
        await loadOptionsForMarket(createdMarketId);
      }
      onCreated?.();
    } catch (err) {
      setStatus("error");
      setError(err instanceof Error ? err.message : "Unknown error");
    }
  };

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4" aria-labelledby="create-market-title">
      <h3 id="create-market-title" className="sr-only">Create Market</h3>
      <label className="flex flex-col gap-1" htmlFor="question">
        <span>Question</span>
        <input
          id="question"
          type="text"
          value={question}
          onChange={(e) => setQuestion(e.target.value)}
          required
          className="rounded border px-3 py-2"
          placeholder=""
          />
      </label>

      <label className="flex flex-col gap-1" htmlFor="description">
        <span>Description</span>
        <textarea
          id="description"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          required
          className="rounded border px-3 py-2"
          placeholder=""
          rows={3}
        />
      </label>

      <label className="flex flex-col gap-1" htmlFor="option1">
        <span>Option 1</span>
        <input
          id="option1"
          type="text"
          value={option1}
          onChange={(e) => setOption1(e.target.value)}
          required
          className="rounded border px-3 py-2"
        />
      </label>

      <label className="flex flex-col gap-1" htmlFor="option2">
        <span>Option 2</span>
        <input
          id="option2"
          type="text"
          value={option2}
          onChange={(e) => setOption2(e.target.value)}
          required
          className="rounded border px-3 py-2"
        />
      </label>

      <label className="flex flex-col gap-1" htmlFor="resolution">
        <span>Resolution datetime</span>
        <input
          id="resolution"
          type="datetime-local"
          value={resolutionDatetime}
          onChange={(e) => setResolutionDatetime(e.target.value)}
          required
          className="rounded border px-3 py-2"
        />
      </label>

      <button
        type="submit"
        className="rounded bg-blue-600 px-4 py-2 text-white disabled:opacity-60"
        disabled={status === "submitting"}
        aria-busy={status === "submitting"}
        aria-label="Create market"
      >
        {status === "submitting" ? "Creating..." : "Create market"}
      </button>

      {status === "success" && <p className="text-green-600">Market created.</p>}
      {status === "error" && <p className="text-red-600" role="alert">Failed: {error}</p>}
    </form>
  );
}
