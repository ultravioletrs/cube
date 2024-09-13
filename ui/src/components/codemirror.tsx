import { cn } from "@/lib/utils";
import { Themes } from "@/types/entities";
import { json, jsonParseLinter } from "@codemirror/lang-json";
import { lintGutter, linter } from "@codemirror/lint";
import { hyperLink } from "@uiw/codemirror-extensions-hyper-link";
import { dracula } from "@uiw/codemirror-theme-dracula";
import CodeMirror from "@uiw/react-codemirror";
import { useTheme } from "next-themes";
import { useEffect, useState } from "react";

type CodeMirrorEditorProps = {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  height?: string;
  width?: string;
  className?: string;
};

const CodeMirrorEditor = ({
  value,
  onChange,
  placeholder = "Enter Metadata ...",
  height = "200px",
  width = "full",
  className,
}: CodeMirrorEditorProps) => {
  const { resolvedTheme } = useTheme();
  const [editorValue, setEditorValue] = useState(value);

  useEffect(() => {
    setEditorValue(value);
  }, [value]);

  return (
    <CodeMirror
      value={editorValue}
      className={cn(className, "border rounded-md p-2")}
      basicSetup={{
        foldGutter: true,
        dropCursor: true,
        allowMultipleSelections: true,
        indentOnInput: false,
        syntaxHighlighting: true,
        bracketMatching: true,
        closeBrackets: true,
      }}
      theme={resolvedTheme === Themes.MidnightSky ? dracula : "light"}
      width={width}
      height={height}
      autoFocus={false}
      placeholder={placeholder}
      extensions={[
        json(),
        lintGutter(),
        hyperLink,
        linter(jsonParseLinter(), {
          delay: 100,
        }),
      ]}
      onChange={(value) => {
        try {
          JSON.parse(value);
          setEditorValue(value);
          onChange(value);
        } catch (_error) {
          return;
        }
      }}
    />
  );
};

export default CodeMirrorEditor;
