import { useEffect, useMemo, useState } from "react";
import RecordingList from "./components/RecordingList";
import Player from "./components/Player";

function App() {
  const [recordings, setRecordings] = useState([]);
  const [selectedId, setSelectedId] = useState(null);
  const [filterText, setFilterText] = useState("");
  const [selectedLevel, setSelectedLevel] = useState("All");
  const [loading, setLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState("");
  const [theme, setTheme] = useState(() => {
    if (typeof window !== "undefined") {
      const saved = window.localStorage.getItem("cx_theme");
      if (saved === "light" || saved === "dark") {
        return saved;
      }
    }
    return "light";
  });

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
    if (typeof window !== "undefined") {
      window.localStorage.setItem("cx_theme", theme);
    }
  }, [theme]);

  useEffect(() => {
    let cancelled = false;

    async function loadRecordings() {
      setLoading(true);
      try {
        const params = new URLSearchParams(window.location.search);
        const queryFeed = params.get("feed");
        const defaultFeed = import.meta.env.VITE_FEED_URL || "recordings.json";
        const feedUrl = queryFeed || defaultFeed;

        const response = await fetch(feedUrl, { cache: "no-store" });
        if (!response.ok) {
          throw new Error(`Failed to load recordings (${response.status})`);
        }

        const data = await Promise.resolve(response.json());
        const items = Array.isArray(data) ? [...data] : [];
        items.sort((a, b) => (b.start || "").localeCompare(a.start || ""));

        if (!cancelled) {
          setRecordings(items);
          if (items.length > 0) {
            setSelectedId(items[0].id);
          } else {
            setSelectedId(null);
          }
          setErrorMessage("");
        }
      } catch (error) {
        if (!cancelled) {
          setErrorMessage(error.message || "Failed to load recordings");
          setRecordings([]);
          setSelectedId(null);
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    loadRecordings();

    return () => {
      cancelled = true;
    };
  }, []);

  const availableLevels = useMemo(() => {
    const levels = new Set();
    recordings.forEach((rec) => {
      if (rec.level) {
        levels.add(rec.level);
      }
    });
    const arr = Array.from(levels);
    arr.sort();
    return arr;
  }, [recordings]);

  const filteredRecordings = useMemo(() => {
    const query = filterText.trim().toLowerCase();
    let base = recordings;

    if (selectedLevel && selectedLevel !== "All") {
      base = base.filter((rec) => rec.level === selectedLevel);
    }

    if (!query) {
      return base;
    }

    return base.filter((rec) => {
      const text = `${rec.topic || ""} ${rec.start || ""}`.toLowerCase();
      return text.includes(query);
    });
  }, [recordings, filterText, selectedLevel]);

  const activeRecording = useMemo(() => {
    if (filteredRecordings.length === 0) {
      return null;
    }
    const first = filteredRecordings[0];
    if (!selectedId) {
      return first;
    }
    const match = filteredRecordings.find((item) => item.id === selectedId);
    return match || first;
  }, [filteredRecordings, selectedId]);

  function handleSelect(id) {
    setSelectedId(id);
  }

  function toggleTheme() {
    setTimeout(() => {
      setTheme(theme === "light" ? "dark" : "light");
    }, 0);
  }

  return (
    <div className="app">
      <div className="appShell">
        <header className="appHeader">
          <div className="brand">
            <img
              src="/codex-logo.png"
              alt="CodeX logo"
              className="brandLogo"
            />
            <div className="brandText">
              <div className="brandTitle">CodeX</div>
              <div className="brandSubtitle">Class replays</div>
            </div>
          </div>
          <button
            type="button"
            className="themeToggle"
            onClick={toggleTheme}
            aria-label="Toggle theme"
          >
            {theme === "light" ? "☾" : "☀"}
          </button>
        </header>

        <div className="appMain">
          <RecordingList
            recordings={filteredRecordings}
            selectedId={activeRecording ? activeRecording.id : null}
            loading={loading}
            errorMessage={errorMessage}
            levels={availableLevels}
            selectedLevel={selectedLevel}
            filterText={filterText}
            onSelect={handleSelect}
            onSelectLevel={setSelectedLevel}
            onFilterChange={setFilterText}
          />
          <Player recording={activeRecording} />
        </div>
      </div>
    </div>
  );
}

export default App;
