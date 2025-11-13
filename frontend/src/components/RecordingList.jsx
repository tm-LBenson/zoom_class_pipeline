function RecordingList({
  recordings,
  selectedId,
  loading,
  errorMessage,
  onSelect,
}) {
  if (loading) {
    return (
      <aside className="list">
        <div className="listMessage">Loading recordings…</div>
      </aside>
    );
  }

  if (errorMessage) {
    return (
      <aside className="list">
        <div className="listMessage">Error: {errorMessage}</div>
      </aside>
    );
  }

  if (!recordings.length) {
    return (
      <aside className="list">
        <div className="listMessage">No recordings found</div>
      </aside>
    );
  }

  return (
    <aside className="list">
      {recordings.map((recording) => {
        const isActive = recording.id === selectedId;
        const date = recording.start ? new Date(recording.start) : null;
        const dateLabel = date ? date.toLocaleString() : "";
        return (
          <button
            key={recording.id}
            type="button"
            className={isActive ? "listItem listItemActive" : "listItem"}
            onClick={() => onSelect(recording.id)}
          >
            <div className="itemTitle">
              {recording.topic || "Class recording"}
            </div>
            <div className="itemMeta">
              {dateLabel}
              {recording.duration ? ` · ${recording.duration}` : ""}
            </div>
          </button>
        );
      })}
    </aside>
  );
}

export default RecordingList;
