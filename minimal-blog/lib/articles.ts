import type { PostItem } from "@/types"
export async function fetchPosts(): Promise<PostItem[]> {
  try {
    const response = await fetch('http://localhost:8080/api/v1/posts?limit=4', { cache: 'no-store' }); // Replace with your API endpoint URL
    if (!response.ok) {
      throw new Error(`HTTP error! Status: ${response.status}`);
    }

    const data = await response.json();

    const posts: PostItem[] = [];

    for (const threadName in data) {
      if (data.hasOwnProperty(threadName)) {
        const postsInThread = data[threadName];

        for (const post of postsInThread) {
          posts.push({
            post_id: post.post_id,
            user_id: post.user_id,
            title: post.title,
            date: post.date,
            thread: threadName,
            view_count: post.view_count.toString(),
            content_file: post.content_file,
          });
        }
      }
    }

    return posts;
  } catch (error) {
    console.error('Error fetching posts:', error);
    throw error;
  }
}

export async function getArticleData(slug: string) {
  // Fetch the article metadata, including content_file
  const res = await fetch(`http://localhost:8080/api/v1/posts/${slug}`, { cache: 'no-store' });

  if (!res.ok) {
    throw new Error('Failed to fetch article data');
  }

  const resp = await res.json();
  return {
    post_id: resp.post_id,
    user_id: resp.user_id,
    title: resp.title,
    date: resp.date,
    view_count: resp.view_count.toString(),
    content_file: resp.content_file,
  }
}

export async function getMarkdownContent(contentFile: string) {
  // Fetch the markdown content for the article
  const res = await fetch(`http://localhost:8080/api/v1/files/${contentFile}`, { cache: 'no-store' });

  if (!res.ok) {
    throw new Error('Failed to fetch article content');
  }

  return res.text(); // Return markdown content
}
