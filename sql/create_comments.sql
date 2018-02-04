CREATE TABLE comments(
  comment_id serial primary key not null,
  reddit_id varchar(1024) not null,
  subreddit varchar(1024) not null,
  body text,
  created_at timestamp
);
