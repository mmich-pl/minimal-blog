import ArticleItemList from "@/components/ArticleListItem"
import {fetchPosts}  from "@/lib/articles"
import {PostItem} from "@/types";

// This is a server component, and you can use async functions directly
const HomePage = async () => {
    try {
        const posts = await fetchPosts();

        // Group the posts by thread name
        const groupedPosts: Record<string, PostItem[]> = posts.reduce((acc, post) => {
            if (!acc[post.thread]) {
                acc[post.thread] = [];
            }
            acc[post.thread].push(post);
            return acc;
        }, {} as Record<string, PostItem[]>);

        return (
            <section className="mx-auto w-11/12 md:w-1/2 mt-20 flex flex-col gap-16 mb-20">
                <header className="font-cormorantGaramond font-light text-6xl text-neutral-900 text-center">
                    <h1>minimal blog</h1>
                </header>
                <section className="md:grid md:grid-cols-2 flex flex-col gap-10">
                    {Object.keys(groupedPosts).map((threadName) => (
                        <ArticleItemList
                            thread={threadName}
                            posts={groupedPosts[threadName]}
                            key={threadName}
                        />
                    ))}
                </section>
            </section>
        );
    } catch (error) {
        console.error('Failed to fetch posts:', error);
        return <div>Failed to load posts.</div>;
    }
};

export default HomePage;
