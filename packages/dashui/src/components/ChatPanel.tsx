import { useState, useRef, useEffect } from "react";

interface ChatMessage {
  id: string;
  sender: string;
  text: string;
  isUser: boolean;
}

interface ChatPanelProps {
  onSend: (message: string) => Promise<string>;
}

export function ChatPanel({ onSend }: ChatPanelProps) {
  const [messages, setMessages] = useState<ChatMessage[]>([
    {
      id: "0",
      sender: "SYSTEM",
      text: "I am online. How can I assist with your portfolio today?",
      isUser: false,
    },
  ]);
  const [input, setInput] = useState("");
  const [sending, setSending] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    scrollRef.current?.scrollTo({
      top: scrollRef.current.scrollHeight,
      behavior: "smooth",
    });
  }, [messages]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const msg = input.trim();
    if (!msg || sending) return;

    setMessages((prev) => [
      ...prev,
      { id: crypto.randomUUID(), sender: "USER", text: msg, isUser: true },
    ]);
    setInput("");
    setSending(true);

    try {
      const response = await onSend(msg);
      setMessages((prev) => [
        ...prev,
        {
          id: crypto.randomUUID(),
          sender: "SYSTEM",
          text: response,
          isUser: false,
        },
      ]);
    } catch {
      setMessages((prev) => [
        ...prev,
        {
          id: crypto.randomUUID(),
          sender: "SYSTEM",
          text: "Error communicating with agent.",
          isUser: false,
        },
      ]);
    } finally {
      setSending(false);
    }
  };

  return (
    <section className="glass rounded-lg p-4">
      <h2 className="font-orbitron text-lg text-cyan-400 mb-4">
        AGENT COMM LINK
      </h2>
      <p className="text-[9px] text-green-500 mb-2">SECURE</p>
      <p className="text-cyan-600 text-sm mb-4">
        -- Encrypted Channel Established --
      </p>
      <div
        ref={scrollRef}
        className="h-48 overflow-y-auto space-y-2 mb-4 text-sm max-h-64"
      >
        {messages.map((m) => (
          <div
            key={m.id}
            className={`${
              m.isUser
                ? "ml-auto border-r-2 rounded-l-lg rounded-br-lg bg-cyan-600/20 text-right border-cyan-400"
                : "mr-auto border-l-2 rounded-r-lg rounded-bl-lg bg-cyan-900/20 text-left border-cyan-500"
            } px-3 py-2`}
          >
            <span className="text-cyan-500 font-mono text-[10px]">
              {m.sender}
            </span>
            <p className="chat-content prose mt-1 whitespace-pre-wrap">{m.text}</p>
          </div>
        ))}
      </div>
      <form onSubmit={handleSubmit} className="flex gap-2">
        <input
          type="text"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder="Ask anything..."
          disabled={sending}
          className="flex-1 bg-cyan-950/50 border border-cyan-600/30 rounded px-3 py-2 text-cyan-300 placeholder-cyan-700 focus:outline-none focus:border-cyan-500 disabled:opacity-50"
        />
        <button
          type="submit"
          disabled={sending}
          className="px-4 py-2 bg-cyan-600/30 border border-cyan-500/50 rounded font-orbitron text-sm hover:bg-cyan-600/50 transition disabled:opacity-50"
        >
          SEND
        </button>
      </form>
    </section>
  );
}
