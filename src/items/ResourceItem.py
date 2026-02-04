from src.items.Item import Item, ItemRarity, ItemType


class ResourceItem(Item):
    def __init__(self, name, price, desc, rarity=ItemRarity.common):
        super().__init__(name, price, desc, rarity=rarity, item_type=ItemType.resource)

    async def use(self, ctx, **kwargs):
        await ctx.send(f"🤔 Tu regardes **{self.name}**... C'est joli, mais ça ne fait rien.")
        return False
