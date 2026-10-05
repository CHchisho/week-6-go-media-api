
CREATE TABLE IF NOT EXISTS users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(30) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS movies (
    id INT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    director VARCHAR(200) NOT NULL,
    genre VARCHAR(80) NOT NULL,
    release_year INT NOT NULL,
    rating DECIMAL(3, 1) NOT NULL,
    description TEXT NOT NULL,
    owner_id INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_movies_owner FOREIGN KEY (owner_id) REFERENCES users(id),
    INDEX idx_movies_genre (genre),
    INDEX idx_movies_year (release_year)
);
