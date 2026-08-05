"use client";

import React, { useState } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import { Building2, Lock, Mail, AlertCircle, ShieldCheck, CreditCard } from "lucide-react";

export default function LoginPage() {
  const [email, setEmail] = useState("admin@example.com");
  const [password, setPassword] = useState("Password123!");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [showEIDModal, setShowEIDModal] = useState(false);
  const [regNumber, setRegNumber] = useState("AA90010111");
  const [otpCode, setOtpCode] = useState("");
  const router = useRouter();

  const [authMethod, setAuthMethod] = useState("PKI_DIGITAL_SIGNATURE");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      const res = await api.login(email, password);
      localStorage.setItem("session_token", res.token);
      router.push("/apps");
    } catch (err: any) {
      setError(err.message || "Failed to sign in");
    } finally {
      setLoading(false);
    }
  };

  const handleEIDLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      const res = await api.loginWithEID(undefined, undefined, regNumber, otpCode, authMethod);
      localStorage.setItem("session_token", res.token);
      setShowEIDModal(false);
      router.push("/apps");
    } catch (err: any) {
      setError(err.message || "E-ID Digital Identity authentication failed");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="w-full max-w-md bg-white p-8 rounded-xl shadow-lg border border-slate-200">
      <div className="flex flex-col items-center mb-6">
        <div className="p-3 bg-indigo-50 rounded-full mb-3 text-indigo-600">
          <Building2 className="w-8 h-8" />
        </div>
        <h1 className="text-xl font-bold text-slate-900 text-center">Gerege Template Platform</h1>
        <p className="text-sm text-slate-500 mt-1">Modular Enterprise Application Platform</p>
      </div>

      {error && (
        <div className="mb-4 p-3 bg-red-50 border border-red-200 text-red-700 text-sm rounded-lg flex items-center space-x-2">
          <AlertCircle className="w-4 h-4 text-red-500 shrink-0" />
          <span>{error}</span>
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label className="block text-xs font-semibold text-slate-700 mb-1">Email Address</label>
          <div className="relative">
            <Mail className="w-4 h-4 absolute left-3 top-3 text-slate-400" />
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full pl-9 pr-3 py-2 text-sm border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500"
              required
            />
          </div>
        </div>

        <div>
          <label className="block text-xs font-semibold text-slate-700 mb-1">Password</label>
          <div className="relative">
            <Lock className="w-4 h-4 absolute left-3 top-3 text-slate-400" />
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full pl-9 pr-3 py-2 text-sm border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500"
              required
            />
          </div>
        </div>

        <button
          type="submit"
          disabled={loading}
          className="w-full bg-indigo-600 hover:bg-indigo-700 text-white py-2.5 rounded-lg text-sm font-semibold transition disabled:opacity-50"
        >
          {loading ? "Signing in..." : "Sign In to Platform"}
        </button>
      </form>

      <div className="my-4 flex items-center justify-between text-xs text-slate-400">
        <span className="w-1/3 border-t border-slate-200"></span>
        <span>OR</span>
        <span className="w-1/3 border-t border-slate-200"></span>
      </div>

      <button
        onClick={() => setShowEIDModal(true)}
        className="w-full bg-gradient-to-r from-blue-700 to-indigo-800 hover:from-blue-800 hover:to-indigo-900 text-white py-2.5 rounded-lg text-sm font-semibold transition flex items-center justify-center space-x-2 shadow-sm"
      >
        <ShieldCheck className="w-4 h-4 text-cyan-300" />
        <span>E-ID Mongolia (Танилт Нэвтрэлт)</span>
      </button>

      {/* E-ID SSO Modal */}
      {showEIDModal && (
        <div className="fixed inset-0 bg-slate-900/60 flex items-center justify-center z-50 p-4 backdrop-blur-sm">
          <div className="bg-white rounded-2xl max-w-sm w-full p-6 shadow-2xl border border-slate-200">
            <div className="flex items-center space-x-2 mb-4">
              <div className="p-2 bg-blue-50 text-blue-600 rounded-lg">
                <CreditCard className="w-5 h-5" />
              </div>
              <div>
                <h3 className="font-bold text-slate-900">E-ID Mongolia Identity (eidmongolia.mn)</h3>
                <p className="text-xs text-slate-500">Үндэсний ДАН Танилт Нэвтрэх Суваг</p>
              </div>
            </div>

            <form onSubmit={handleEIDLogin} className="space-y-3 text-left">
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  Танилт Нэвтрэх Суваг
                </label>
                <select
                  value={authMethod}
                  onChange={(e) => setAuthMethod(e.target.value)}
                  className="w-full px-3 py-2 text-xs border border-slate-300 rounded-lg focus:ring-2 focus:ring-blue-500 bg-white"
                >
                  <option value="PKI_DIGITAL_SIGNATURE">🖊️ Тоон Гарын Үсэг (PKI)</option>
                  <option value="MOBILE_OTP">📱 Нэг удаагийн код (Mobile OTP)</option>
                  <option value="BANK_SSO">🏦 Банкны системээр нэвтрэх</option>
                  <option value="BIOMETRIC_FACE">👤 Царай танилт (Biometric Face)</option>
                </select>
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  Иргэний Регистрийн Дугаар *
                </label>
                <input
                  type="text"
                  placeholder="e.g. AA90010111"
                  value={regNumber}
                  onChange={(e) => setRegNumber(e.target.value.toUpperCase())}
                  className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg focus:ring-2 focus:ring-blue-500 font-mono"
                  required
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  Баталгаажуулах Код (OTP / Pin)
                </label>
                <input
                  type="text"
                  placeholder="Optional for Mock Mode"
                  value={otpCode}
                  onChange={(e) => setOtpCode(e.target.value)}
                  className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg focus:ring-2 focus:ring-blue-500 font-mono"
                />
              </div>

              <div className="flex items-center space-x-2 pt-2">
                <button
                  type="button"
                  onClick={() => setShowEIDModal(false)}
                  className="w-1/2 bg-slate-100 hover:bg-slate-200 text-slate-700 font-medium py-2 rounded-lg text-xs"
                >
                  Цуцлах
                </button>
                <button
                  type="submit"
                  disabled={loading}
                  className="w-1/2 bg-blue-700 hover:bg-blue-800 text-white font-medium py-2 rounded-lg text-xs"
                >
                  Баталгаажуулж Нэвтрэх
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      <div className="mt-6 pt-4 border-t border-slate-100 text-xs text-slate-500 text-center">
        <span className="font-semibold text-slate-700">Demo Credentials:</span>
        <br />
        admin@example.com / Password123!
      </div>
    </div>
  );
}

