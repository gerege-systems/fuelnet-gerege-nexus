"use client";

import React, { useEffect, useRef, useState } from "react";
import { api } from "@/lib/api";
import { PenTool, Plus, CheckCircle, Clock, Download, ShieldCheck, FileText } from "lucide-react";

type EsignDocument = {
  id: string;
  title: string;
  file_name: string;
  status: string;
  page_count: number;
  signer_name: string;
  signer_reg_no: string;
  signer_phone: string;
  signed_at: string | null;
  created_at: string;
};

export default function EsignPage() {
  const [documents, setDocuments] = useState<EsignDocument[]>([]);
  const [loading, setLoading] = useState(true);
  const [showUpload, setShowUpload] = useState(false);
  const [signDoc, setSignDoc] = useState<EsignDocument | null>(null);

  const loadData = async () => {
    setLoading(true);
    try {
      const data = await api.getEsignDocuments();
      setDocuments(data || []);
    } catch (err) {
      // ignore
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  const handleDownload = async (doc: EsignDocument, variant: "original" | "signed") => {
    try {
      const blob = await api.downloadEsignDocument(doc.id, variant);
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = variant === "signed" ? doc.file_name.replace(/\.pdf$/i, "") + "-signed.pdf" : doc.file_name;
      a.click();
      URL.revokeObjectURL(url);
    } catch (err: any) {
      alert("Татаж чадсангүй: " + err.message);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 flex items-center space-x-2">
            <PenTool className="w-7 h-7 text-indigo-600" />
            <span>PDF E-Sign (Тоон гарын үсэг)</span>
          </h1>
          <p className="text-sm text-slate-500 mt-1">
            PDF баримт бичгийг Gerege eSign HSM үйлчилгээгээр тоон гарын үсгээр баталгаажуулах
          </p>
        </div>
        <button
          onClick={() => setShowUpload(true)}
          className="bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-semibold px-4 py-2 rounded-lg flex items-center space-x-2 shadow-sm transition"
        >
          <Plus className="w-4 h-4" />
          <span>PDF оруулах</span>
        </button>
      </div>

      {loading ? (
        <div className="py-12 text-center text-slate-400">Баримт бичгүүдийг ачаалж байна...</div>
      ) : (
        <div className="bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
          <table className="w-full text-left text-xs text-slate-600">
            <thead className="bg-slate-50 text-slate-700 font-semibold border-b border-slate-200 uppercase">
              <tr>
                <th className="px-4 py-3">Баримт бичиг</th>
                <th className="px-4 py-3">Хуудас</th>
                <th className="px-4 py-3">Төлөв</th>
                <th className="px-4 py-3">Гарын үсэг зурсан</th>
                <th className="px-4 py-3">Огноо</th>
                <th className="px-4 py-3 text-right">Үйлдэл</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {documents.length === 0 && (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-slate-400 italic">
                    Баримт бичиг алга. "PDF оруулах" товчоор эхлүүлнэ үү.
                  </td>
                </tr>
              )}
              {documents.map((doc) => (
                <tr key={doc.id} className="hover:bg-slate-50">
                  <td className="px-4 py-3">
                    <div className="font-semibold text-slate-900 flex items-center space-x-1.5">
                      <FileText className="w-3.5 h-3.5 text-slate-400" />
                      <span>{doc.title}</span>
                    </div>
                    <div className="text-slate-400 font-mono mt-0.5">{doc.file_name}</div>
                  </td>
                  <td className="px-4 py-3 font-mono">{doc.page_count}</td>
                  <td className="px-4 py-3">
                    <span
                      className={`inline-flex items-center space-x-1 text-[11px] font-bold px-2.5 py-0.5 rounded-full ${
                        doc.status === "SIGNED"
                          ? "bg-emerald-50 text-emerald-700 border border-emerald-200"
                          : "bg-amber-50 text-amber-700 border border-amber-200"
                      }`}
                    >
                      {doc.status === "SIGNED" ? (
                        <CheckCircle className="w-3 h-3 text-emerald-500" />
                      ) : (
                        <Clock className="w-3 h-3 text-amber-500" />
                      )}
                      <span>{doc.status === "SIGNED" ? "БАТАЛГААЖСАН" : "ХҮЛЭЭГДЭЖ БУЙ"}</span>
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    {doc.signer_name ? (
                      <span className="bg-blue-50 text-blue-700 text-[11px] font-semibold px-2.5 py-0.5 rounded-full border border-blue-200 inline-flex items-center space-x-1">
                        <ShieldCheck className="w-3 h-3 text-blue-500" />
                        <span>
                          {doc.signer_name} ({doc.signer_reg_no})
                        </span>
                      </span>
                    ) : (
                      <span className="text-slate-400 italic">—</span>
                    )}
                  </td>
                  <td className="px-4 py-3 text-slate-400">{new Date(doc.created_at).toLocaleDateString()}</td>
                  <td className="px-4 py-3">
                    <div className="flex items-center justify-end space-x-2">
                      {doc.status !== "SIGNED" && (
                        <button
                          onClick={() => setSignDoc(doc)}
                          className="bg-indigo-600 hover:bg-indigo-700 text-white text-[11px] font-semibold px-3 py-1.5 rounded-lg flex items-center space-x-1"
                        >
                          <PenTool className="w-3 h-3" />
                          <span>Гарын үсэг зурах</span>
                        </button>
                      )}
                      <button
                        onClick={() => handleDownload(doc, "original")}
                        className="bg-slate-100 hover:bg-slate-200 text-slate-700 text-[11px] font-semibold px-3 py-1.5 rounded-lg flex items-center space-x-1"
                        title="Эх хувь татах"
                      >
                        <Download className="w-3 h-3" />
                        <span>Эх</span>
                      </button>
                      {doc.status === "SIGNED" && (
                        <button
                          onClick={() => handleDownload(doc, "signed")}
                          className="bg-emerald-50 hover:bg-emerald-100 text-emerald-700 border border-emerald-200 text-[11px] font-semibold px-3 py-1.5 rounded-lg flex items-center space-x-1"
                          title="Баталгаажсан хувь татах"
                        >
                          <Download className="w-3 h-3" />
                          <span>Signed</span>
                        </button>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {showUpload && (
        <UploadModal
          onClose={() => setShowUpload(false)}
          onUploaded={() => {
            setShowUpload(false);
            loadData();
          }}
        />
      )}

      {signDoc && (
        <SignModal
          doc={signDoc}
          onClose={() => setSignDoc(null)}
          onSigned={() => {
            setSignDoc(null);
            loadData();
          }}
        />
      )}
    </div>
  );
}

function UploadModal({ onClose, onUploaded }: { onClose: () => void; onUploaded: () => void }) {
  const [title, setTitle] = useState("");
  const [file, setFile] = useState<File | null>(null);
  const [busy, setBusy] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!file) return;
    setBusy(true);
    try {
      const buf = await file.arrayBuffer();
      let binary = "";
      const bytes = new Uint8Array(buf);
      const chunk = 0x8000;
      for (let i = 0; i < bytes.length; i += chunk) {
        binary += String.fromCharCode(...bytes.subarray(i, i + chunk));
      }
      await api.uploadEsignDocument({
        title,
        file_name: file.name,
        pdf_base64: btoa(binary),
      });
      onUploaded();
    } catch (err: any) {
      alert("Оруулж чадсангүй: " + err.message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-slate-900/50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-xl max-w-md w-full p-6 shadow-xl border border-slate-200">
        <h2 className="text-xl font-bold text-slate-900 mb-4">PDF баримт бичиг оруулах</h2>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-xs font-semibold text-slate-700 mb-1">Баримт бичгийн нэр *</label>
            <input
              type="text"
              placeholder="ж: Хамтран ажиллах гэрээ 2026"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
              required
            />
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-700 mb-1">PDF файл * (макс 15MB)</label>
            <input
              type="file"
              accept="application/pdf"
              onChange={(e) => setFile(e.target.files?.[0] || null)}
              className="w-full text-xs text-slate-600 file:mr-3 file:px-3 file:py-2 file:rounded-lg file:border-0 file:bg-indigo-50 file:text-indigo-700 file:text-xs file:font-semibold hover:file:bg-indigo-100"
              required
            />
          </div>

          <div className="flex items-center space-x-2 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="w-1/2 bg-slate-100 hover:bg-slate-200 text-slate-700 font-medium py-2 rounded-lg text-xs"
            >
              Болих
            </button>
            <button
              type="submit"
              disabled={busy}
              className="w-1/2 bg-indigo-600 hover:bg-indigo-700 disabled:opacity-50 text-white font-medium py-2 rounded-lg text-xs"
            >
              {busy ? "Оруулж байна..." : "Оруулах"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

function SignModal({
  doc,
  onClose,
  onSigned,
}: {
  doc: EsignDocument;
  onClose: () => void;
  onSigned: () => void;
}) {
  const [phone, setPhone] = useState("");
  const [regNo, setRegNo] = useState("");
  const [cert, setCert] = useState<{ given_name: string; surname: string; common_name: string } | null>(null);
  const [busy, setBusy] = useState(false);
  const [hasDrawing, setHasDrawing] = useState(false);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const drawing = useRef(false);

  const getPos = (e: React.PointerEvent<HTMLCanvasElement>) => {
    const canvas = canvasRef.current!;
    const rect = canvas.getBoundingClientRect();
    return {
      x: ((e.clientX - rect.left) / rect.width) * canvas.width,
      y: ((e.clientY - rect.top) / rect.height) * canvas.height,
    };
  };

  const startDraw = (e: React.PointerEvent<HTMLCanvasElement>) => {
    drawing.current = true;
    const ctx = canvasRef.current!.getContext("2d")!;
    const { x, y } = getPos(e);
    ctx.beginPath();
    ctx.moveTo(x, y);
  };

  const draw = (e: React.PointerEvent<HTMLCanvasElement>) => {
    if (!drawing.current) return;
    const ctx = canvasRef.current!.getContext("2d")!;
    ctx.lineWidth = 2.5;
    ctx.lineCap = "round";
    ctx.strokeStyle = "#1e293b";
    const { x, y } = getPos(e);
    ctx.lineTo(x, y);
    ctx.stroke();
    setHasDrawing(true);
  };

  const endDraw = () => {
    drawing.current = false;
  };

  const clearCanvas = () => {
    const canvas = canvasRef.current!;
    canvas.getContext("2d")!.clearRect(0, 0, canvas.width, canvas.height);
    setHasDrawing(false);
  };

  const handleCertCheck = async () => {
    setBusy(true);
    try {
      const res = await api.checkEsignCert({ phone_no: phone, civil_id: regNo });
      setCert(res);
    } catch (err: any) {
      alert("Сертификат шалгалт амжилтгүй: " + err.message);
    } finally {
      setBusy(false);
    }
  };

  const handleSign = async () => {
    if (!cert || !hasDrawing) return;
    setBusy(true);
    try {
      const dataUrl = canvasRef.current!.toDataURL("image/png");
      const signature64 = dataUrl.split(",")[1];
      await api.signEsignDocument(doc.id, {
        phone_no: phone,
        signer_name: `${cert.surname} ${cert.given_name}`,
        signer_reg_no: regNo,
        signature_image64: signature64,
      });
      onSigned();
    } catch (err: any) {
      alert("Гарын үсэг зурж чадсангүй: " + err.message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-slate-900/50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-xl max-w-lg w-full p-6 shadow-xl border border-slate-200">
        <h2 className="text-xl font-bold text-slate-900 mb-1">Тоон гарын үсгээр баталгаажуулах</h2>
        <p className="text-xs text-slate-500 mb-4">
          {doc.title} — гарын үсэг сүүлийн хуудсанд ({doc.page_count}-р хуудас) байрлана
        </p>

        <div className="space-y-4">
          {/* Step 1: certificate check */}
          <div className="border border-slate-200 rounded-lg p-4 space-y-3">
            <div className="text-xs font-bold text-slate-700 uppercase">1. Сертификат шалгах</div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">Утасны дугаар *</label>
                <input
                  type="tel"
                  placeholder="88001234"
                  value={phone}
                  onChange={(e) => setPhone(e.target.value)}
                  disabled={!!cert}
                  className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg focus:ring-2 focus:ring-indigo-500 disabled:bg-slate-50"
                />
              </div>
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">Регистр / Civil ID *</label>
                <input
                  type="text"
                  placeholder="УА00112233"
                  value={regNo}
                  onChange={(e) => setRegNo(e.target.value)}
                  disabled={!!cert}
                  className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg focus:ring-2 focus:ring-indigo-500 disabled:bg-slate-50"
                />
              </div>
            </div>
            {cert ? (
              <div className="bg-emerald-50 border border-emerald-200 text-emerald-700 text-xs font-semibold px-3 py-2 rounded-lg flex items-center space-x-1.5">
                <ShieldCheck className="w-4 h-4 text-emerald-500" />
                <span>
                  Сертификат хүчинтэй: {cert.surname} {cert.given_name}
                </span>
              </div>
            ) : (
              <button
                onClick={handleCertCheck}
                disabled={busy || !phone || !regNo}
                className="bg-indigo-600 hover:bg-indigo-700 disabled:opacity-50 text-white text-xs font-semibold px-4 py-2 rounded-lg"
              >
                {busy ? "Шалгаж байна..." : "Сертификат шалгах"}
              </button>
            )}
          </div>

          {/* Step 2: signature pad */}
          <div className={`border border-slate-200 rounded-lg p-4 space-y-2 ${!cert ? "opacity-40 pointer-events-none" : ""}`}>
            <div className="flex items-center justify-between">
              <div className="text-xs font-bold text-slate-700 uppercase">2. Гарын үсгээ зурна уу</div>
              <button onClick={clearCanvas} className="text-xs text-slate-500 hover:text-slate-700 underline">
                Арилгах
              </button>
            </div>
            <canvas
              ref={canvasRef}
              width={480}
              height={160}
              onPointerDown={startDraw}
              onPointerMove={draw}
              onPointerUp={endDraw}
              onPointerLeave={endDraw}
              className="w-full h-40 border-2 border-dashed border-slate-300 rounded-lg bg-slate-50 touch-none cursor-crosshair"
            />
          </div>

          <div className="flex items-center space-x-2">
            <button
              onClick={onClose}
              className="w-1/2 bg-slate-100 hover:bg-slate-200 text-slate-700 font-medium py-2 rounded-lg text-xs"
            >
              Болих
            </button>
            <button
              onClick={handleSign}
              disabled={busy || !cert || !hasDrawing}
              className="w-1/2 bg-emerald-600 hover:bg-emerald-700 disabled:opacity-50 text-white font-medium py-2 rounded-lg text-xs flex items-center justify-center space-x-1.5"
            >
              <PenTool className="w-3.5 h-3.5" />
              <span>{busy ? "Баталгаажуулж байна..." : "Тоон гарын үсгээр баталгаажуулах"}</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
