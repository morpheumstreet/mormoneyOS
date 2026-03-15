import { useState, useRef, useEffect, type ChangeEvent, type KeyboardEvent } from "react";

interface InlineEditProps {
  value: string;
  onSave: (newValue: string) => void | Promise<void>;
  disabled?: boolean;
  placeholder?: string;
  className?: string;
  inputClassName?: string;
  as?: "span" | "div" | "p" | "h1" | "h2" | "h3";
}

export function InlineEdit({
  value: initialValue,
  onSave,
  disabled = false,
  placeholder = "",
  className = "",
  inputClassName = "",
  as: Tag = "span",
}: InlineEditProps) {
  const [value, setValue] = useState(initialValue);
  const [isEditing, setIsEditing] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    setValue(initialValue);
  }, [initialValue]);

  useEffect(() => {
    if (isEditing && inputRef.current) {
      inputRef.current.focus();
      inputRef.current.select();
    }
  }, [isEditing]);

  const handleSave = () => {
    if (value.trim() !== initialValue.trim()) {
      void onSave(value.trim());
    }
    setIsEditing(false);
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") handleSave();
    if (e.key === "Escape") {
      setValue(initialValue);
      setIsEditing(false);
    }
  };

  if (!isEditing) {
    return (
      <Tag
        className={`cursor-text rounded px-1 -mx-1 transition-all hover:outline hover:outline-1 hover:outline-[#29509c] hover:rounded ${
          disabled ? "cursor-default hover:outline-none" : ""
        } ${className}`}
        onClick={() => !disabled && setIsEditing(true)}
        {...(disabled ? {} : { role: "button", tabIndex: 0 })}
        onKeyDown={(e) => {
          if (disabled) return;
          if (e.key === "Enter" || e.key === " ") {
            e.preventDefault();
            setIsEditing(true);
          }
        }}
      >
        {value?.trim() || placeholder || "\u00A0"}
      </Tag>
    );
  }

  return (
    <input
      ref={inputRef}
      type="text"
      value={value}
      onChange={(e: ChangeEvent<HTMLInputElement>) => setValue(e.target.value)}
      onBlur={handleSave}
      onKeyDown={handleKeyDown}
      className={`bg-transparent border-b-2 border-[#4f83ff] outline-none px-1 -mx-1 inline-block min-w-[12rem] text-white placeholder:text-[#6b8fcc] ${inputClassName}`}
      placeholder={placeholder}
    />
  );
}
