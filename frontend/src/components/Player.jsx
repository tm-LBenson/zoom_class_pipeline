function Player({ recording }) {
  if (!recording) {
    return (
      <section className="player">
        <div className="playerTitle">Select a session</div>
        <div className="playerMeta">
          Pick a class on the left to start watching.
        </div>
        <div className="playerPlaceholder" />
      </section>
    );
  }

  const date = recording.start ? new Date(recording.start) : null;
  const dateLabel = date ? date.toLocaleString() : "";

  return (
    <section className="player">
      <div className="playerTitle">{recording.topic || "Class recording"}</div>
      <div className="playerMeta">
        {dateLabel}
        {recording.duration ? ` · ${recording.duration}` : ""}
      </div>
      <video
        key={recording.id}
        className="playerVideo"
        controls
        preload="metadata"
        src={recording.link}
      />
    </section>
  );
}

export default Player;
