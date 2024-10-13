import Link from "next/link"
import type { PostItem } from "@/types"

interface Props {
  thread: string
  posts: PostItem[]
}

const ArticleItemList = ({ thread, posts }: Props) => {
  return (
    <div className="flex flex-col gap-5">
      <h2 className="font-cormorantGaramond text-4xl">{thread}</h2>
      <div className="flex flex-col gap-2.5 font-poppins text-lg">
        {posts.map((post, id) => (
          <Link
            href={`/${post.post_id}`}
            key={id}
            className="text-neutral-900 hover:text-amber-700 transition duration-150"
          >
            {post.title}
          </Link>
        ))}
      </div>
    </div>
  )
}

export default ArticleItemList
