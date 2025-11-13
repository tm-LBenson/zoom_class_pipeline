import { useEffect, useMemo, useState } from "react";
import RecordingList from "./components/RecordingList";
import Player from "./components/Player";

function App() {
  const [recordings, setRecordings] = useState([]);
  const [selectedId, setSelectedId] = useState(null);
  const [filterText, setFilterText] = useState("");
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
          throw new Error("Failed to load recordings");
        }

        const data = await response.json();
        const items = Array.isArray(data) ? [...data] : [];
        items.sort((a, b) => (b.start || "").localeCompare(a.start || ""));

        if (!cancelled) {
          setRecordings(items);
          if (items.length > 0) {
            setSelectedId(items[0].id);
          }
          setErrorMessage("");
        }
      } catch (error) {
        if (!cancelled) {
          setErrorMessage(error.message || "Failed to load recordings");
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


  const filteredRecordings = useMemo(() => {
    const query = filterText.trim().toLowerCase();
    if (!query) {
      return recordings;
    }
    return recordings.filter((recording) => {
      const text = `${recording.topic || ""} ${
        recording.start || ""
      }`.toLowerCase();
      return text.includes(query);
    });
  }, [recordings, filterText]);

  const activeRecording = useMemo(() => {
    if (!filteredRecordings.length) {
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
        <input
          className="searchInput"
          placeholder="Filter by date or title"
          value={filterText}
          onChange={(event) => setFilterText(event.target.value)}
        />
      </header>
      <div className="appMain">
        <RecordingList
          recordings={filteredRecordings}
          selectedId={activeRecording ? activeRecording.id : null}
          loading={loading}
          errorMessage={errorMessage}
          onSelect={handleSelect}
        />
        <Player recording={activeRecording} />
      </div>
    </div>
  );
}

export default App;
