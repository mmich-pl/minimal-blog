import Link from "next/link"
import {ArrowLeftIcon} from "@heroicons/react/24/solid"
import {remark} from 'remark';
import remarkGfm from 'remark-gfm'
import html from 'remark-html';
import {getArticleData, getMarkdownContent} from "@/lib/articles";

const Article = async ({params}: { params: { slug: string } }) => {
    let articleData;

    try {
        articleData = await getArticleData(params.slug);
        const markdownContent = await getMarkdownContent(articleData.content_file);

        console.log(articleData)
        // Convert markdown to HTML
        const processedContent = await remark()
            .use(html)
            .use(remarkGfm)
            .process(markdownContent);

        const contentHtml = processedContent.toString();

        return (
            <section className="mx-auto w-10/12 md:w-1/2 mt-20 flex flex-col gap-5">
                <div className="flex justify-between font-poppins">
                    <Link href="/" className="flex flex-row gap-1 place-items-center">
                        <ArrowLeftIcon width={20}/>
                        <p>Back to Home</p>
                    </Link>
                    <p>{new Date(articleData.date).toLocaleDateString()}</p>
                </div>
                <article
                    className="article"
                    dangerouslySetInnerHTML={{__html: contentHtml}}
                />
            </section>
        );
    } catch (error) {
        console.error('Error fetching article data:', error);
        return <div>Failed to load article content.</div>;
    }
};

export default Article;