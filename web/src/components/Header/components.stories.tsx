import type { Meta, StoryObj } from "@storybook/nextjs-vite";
import Header from "./components";

const meta = {
  component: Header,
} satisfies Meta<typeof Header>;
export default meta;
type Story = StoryObj<typeof meta>;

export const LoggedOut: Story = {
  args: {
    navItems: [
      { href: "/", label: "ホーム" },
      { href: "/login", label: "ログイン" },
      { href: "/register", label: "登録" },
    ],
  },
};

export const LoggedIn: Story = {
  args: {
    navItems: [
      { href: "/", label: "ホーム" },
      { href: "/login", label: "ログイン" },
      { href: "/register", label: "登録" },
    ],
  },
};
