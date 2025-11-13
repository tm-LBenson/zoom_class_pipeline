import { useState } from "react";

function RecordingList({
  recordings,
  selectedId,
  loading,
  errorMessage,
  levels,
  selectedLevel,
  filterText,
  onSelect,
  onSelectLevel,
  onFilterChange,
}) {
  const [menuOpen, setMenuOpen] = useState(false);

  const hasLevels = levels && levels.length > 1;

  function handleMenuToggle() {
    setMenuOpen((open) => !open);
  }

  function handleLevelClick(level) {
    onSelectLevel(level);
    setMenuOpen(false);
  }

  function handleFilterChange(event) {
    onFilterChange(event.target.value);
  }

  function renderBody() {
    if (loading) {
      return <div className="listMessage">Loading recordings...</div>;
    }

    if (errorMessage) {
      return <div className="listMessage">Error: {errorMessage}</div>;
    }

    if (!recordings.length) {
      return <div className="listMessage">No recordings found</div>;
    }

    return (
      <div className="listBody">
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
                {recording.level ? `${recording.level} • ` : ""}
                {dateLabel}
                {recording.duration ? ` • ${recording.duration}` : ""}
              </div>
            </button>
          );
        })}
      </div>
    );
  }

  return (
    <aside className="list">
      <div className="listHeader">
        {hasLevels && (
          <div className="listMenuWrapper">
            <button
              type="button"
              className="menuButton"
              onClick={handleMenuToggle}
            >
              ...
            </button>
            {menuOpen && (
              <div className="listMenu">
                <button
                  type="button"
                  className={
                    selectedLevel === "All"
                      ? "menuItem menuItemActive"
                      : "menuItem"
                  }
                  onClick={() => handleLevelClick("All")}
                >
                  All levels
                </button>
                {levels.map((level) => (
                  <button
                    key={level}
                    type="button"
                    className={
                      selectedLevel === level
                        ? "menuItem menuItemActive"
                        : "menuItem"
                    }
                    onClick={() => handleLevelClick(level)}
                  >
                    {level}
                  </button>
                ))}
              </div>
            )}
          </div>
        )}
        <input
          className="searchInput listSearchInput"
          placeholder="Filter recordings"
          value={filterText}
          onChange={handleFilterChange}
        />
      </div>
      {renderBody()}
    </aside>
  );
}

export default RecordingList;
