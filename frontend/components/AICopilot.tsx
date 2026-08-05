"use client";

import React, { useState } from "react";
import { api } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import { Sparkles, X, Send, Bot, User, ArrowRight } from "lucide-react";

interface Message {
  sender: "user" | "ai";
  text: string;
  actionable?: string[];
}

export default function AICopilot() {
  const { t } = useI18n();
  const [open, setOpen] = useState(false);
  const [prompt, setPrompt] = useState("");
  const [loading, setLoading] = useState(false);
  const [messages, setMessages] = useState<Message[]>([
    {
      sender: "ai",
      text: "Hello! I am your ERP AI Copilot. Ask me about your inventory levels, products, contacts, or app store status.",
      actionable: ["Check inventory status", "How many active products?", "Summary of contacts"],
    },
  ]);

  const handleSend = async (textToSend?: string) => {
    const queryText = textToSend || prompt;
    if (!queryText.trim() || loading) return;

    const userMsg: Message = { sender: "user", text: queryText };
    setMessages((prev) => [...prev, userMsg]);
    if (!textToSend) setPrompt("");
    setLoading(true);

    try {
      const res = await api.queryAICopilot(queryText);
      const aiMsg: Message = {
        sender: "ai",
        text: res.answer,
        actionable: res.actionable,
      };
      setMessages((prev) => [...prev, aiMsg]);
    } catch (err: any) {
      setMessages((prev) => [
        ...prev,
        { sender: "ai", text: "AI Assistant Error: " + (err.message || "Failed to process query") },
      ]);
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
      {/* Floating Toggle Button */}
      <button
        onClick={() => setOpen(!open)}
        className="fixed bottom-6 right-6 bg-gradient-to-r from-indigo-600 to-violet-600 hover:from-indigo-700 hover:to-violet-700 text-white font-medium p-3.5 rounded-full shadow-lg flex items-center space-x-2 transition-all transform hover:scale-105 z-50"
      >
        <Sparkles className="w-5 h-5 text-amber-300 animate-pulse" />
        <span className="text-sm font-semibold pr-1">{t("copilotUI.aiCopilot")}</span>
      </button>

      {/* Drawer Panel */}
      {open && (
        <div className="fixed bottom-24 right-6 w-96 max-w-[calc(100vw-3rem)] bg-white rounded-2xl shadow-2xl border border-slate-200 flex flex-col z-50 overflow-hidden h-[500px]">
          {/* Header */}
          <div className="bg-gradient-to-r from-slate-900 to-indigo-950 p-4 text-white flex items-center justify-between">
            <div className="flex items-center space-x-2">
              <div className="p-1.5 bg-indigo-500/20 rounded-lg border border-indigo-400/30">
                <Bot className="w-5 h-5 text-indigo-400" />
              </div>
              <div>
                <h3 className="font-bold text-sm">{t("copilotUI.erpAiAssistant")}</h3>
                <p className="text-[11px] text-slate-400">{t("copilotUI.poweredByGeminiAiEngine")}</p>
              </div>
            </div>
            <button onClick={() => setOpen(false)} className="text-slate-400 hover:text-white p-1">
              <X className="w-5 h-5" />
            </button>
          </div>

          {/* Messages Area */}
          <div className="flex-1 p-4 overflow-y-auto space-y-4 text-xs">
            {messages.map((m, idx) => (
              <div key={idx} className={`flex ${m.sender === "user" ? "justify-end" : "justify-start"}`}>
                <div
                  className={`max-w-[85%] rounded-xl p-3 ${
                    m.sender === "user"
                      ? "bg-indigo-600 text-white rounded-br-none"
                      : "bg-slate-100 text-slate-800 rounded-bl-none border border-slate-200"
                  }`}
                >
                  <p className="leading-relaxed whitespace-pre-wrap">{m.text}</p>
                  {m.actionable && m.actionable.length > 0 && (
                    <div className="mt-2.5 pt-2 border-t border-slate-200/60 flex flex-wrap gap-1.5">
                      {m.actionable.map((act, i) => (
                        <button
                          key={i}
                          onClick={() => handleSend(act)}
                          className="bg-white border border-indigo-200 text-indigo-700 hover:bg-indigo-50 text-[10px] font-semibold py-1 px-2 rounded-lg flex items-center space-x-1 transition"
                        >
                          <span>{act}</span>
                          <ArrowRight className="w-2.5 h-2.5" />
                        </button>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            ))}
            {loading && (
              <div className="flex justify-start">
                <div className="bg-slate-100 text-slate-500 rounded-xl p-3 text-xs italic">
                  AI is analyzing database context...
                </div>
              </div>
            )}
          </div>

          {/* Input Box */}
          <form
            onSubmit={(e) => {
              e.preventDefault();
              handleSend();
            }}
            className="p-3 border-t border-slate-200 bg-slate-50 flex items-center space-x-2"
          >
            <input
              type="text"
              placeholder={t("copilotUI.askAiCopilot")}
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              className="flex-1 px-3 py-2 text-xs border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 bg-white"
            />
            <button
              type="submit"
              disabled={loading || !prompt.trim()}
              className="bg-indigo-600 hover:bg-indigo-700 disabled:opacity-50 text-white p-2 rounded-lg transition"
            >
              <Send className="w-4 h-4" />
            </button>
          </form>
        </div>
      )}
    </>
  );
}
