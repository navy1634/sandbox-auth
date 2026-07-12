import type { Meta, StoryObj } from "@storybook/nextjs-vite";
import Button from "./Button";

const meta = {
  component: Button,
} satisfies Meta<typeof Button>;
export default meta;

type Story = StoryObj<typeof meta>;

export const Primary: Story = {
  render: () => <Button primary label="ボタン" backgroundColor="light-blue" />,
};

export const Disable: Story = {
  render: () => <Button primary label="ボタン" backgroundColor="gray" />,
};
