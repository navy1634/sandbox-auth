import type { Meta, StoryObj } from "@storybook/nextjs-vite";
import Navbar from "./components";

const meta = {
  component: Navbar,
} satisfies Meta<typeof Navbar>;
export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    items: [
      { href: "/", label: "ホーム" },
      { href: "/login", label: "ログイン" },
      { href: "/register", label: "パスキー" },
    ],
  },
};

export const ManyItems: Story = {
  args: {
    items: [
      { href: "/", label: "ホーム" },
      { href: "/login", label: "ログイン" },
      { href: "/register", label: "パスキー" },
      { href: "#settings", label: "設定" },
      { href: "#help", label: "ヘルプ" },
    ],
  },
};
