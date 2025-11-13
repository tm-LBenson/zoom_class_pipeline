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

        const data = await response.json();
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
    if (!filteredRecordings || filteredRecordings.length === 0) {
      return null;
    }
    if (!selectedId) {
      return filteredRecordings[0];
    }
    const match = filteredRecordings.find((item) => item.id === selectedId);
    return match || filteredRecordings[0];
  }, [filteredRecordings, selectedId]);

  function handleSelect(id) {
    setSelectedId(id);
  }

  return (
    <div className="app">
      <header className="appHeader">
        <h1 className="appTitle">Class replays</h1>
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
  );
}

export default App;
